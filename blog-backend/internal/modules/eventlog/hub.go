package eventlog

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

const (
	// writeWait bounds a single frame write. A browser tab that got suspended
	// must not pin a goroutine forever.
	writeWait = 10 * time.Second
	// pongWait is how long we tolerate silence before assuming the peer is
	// gone; pingPeriod must stay comfortably below it.
	pongWait   = 60 * time.Second
	pingPeriod = 50 * time.Second
	// maxMessageSize caps inbound frames. Clients only send small commands,
	// so anything larger is a bug or an attack.
	maxMessageSize = 4 << 10
	// sendBuffer is how far one client may lag before it is disconnected.
	// Dropping a slow reader is better than letting it stall the publisher.
	sendBuffer = 64
)

// Frame types on the wire. The envelope is deliberately generic so a later
// feature adds a type instead of a second endpoint.
const (
	// FrameHello is sent once on connect and carries the newest event id the
	// server has, so the client can tell whether it missed anything.
	FrameHello = "hello"
	// FrameEvent carries a single *Event.
	FrameEvent = "event"
	// FrameLog carries a single LogLine from the server's own logger.
	FrameLog = "log"
	// FramePing / FramePong is an application-level heartbeat, on top of the
	// protocol-level one. It gives the browser a way to measure liveness
	// without depending on ping frames it cannot observe.
	FramePing = "ping"
	FramePong = "pong"
	// FrameError reports a rejected command back to the sender.
	FrameError = "error"
)

// outbound is a frame the server sends. Payload stays typed so callers can
// hand over any struct.
type outbound struct {
	Type    string `json:"type"`
	Payload any    `json:"payload,omitempty"`
}

// inbound is a frame the server received. The payload is kept raw so each
// command handler decodes its own shape.
type inbound struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

// CommandHandler answers one client→server frame type. A non-nil reply is sent
// back to that client alone, tagged with the same type. This is the extension
// point: registering a command is all it takes to make the socket do something
// new in the other direction.
type CommandHandler func(ctx context.Context, payload json.RawMessage) (any, error)

// upgrader accepts any origin. The socket authenticates with the JWT carried
// in the query string, which a cross-origin page cannot read out of this app's
// localStorage, so the origin header is not what keeps anyone out here. It
// also matches the engine's existing CORS policy.
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 4096,
	CheckOrigin:     func(*http.Request) bool { return true },
}

type client struct {
	hub  *hub
	conn *websocket.Conn
	send chan []byte
}

// hub owns the set of live sockets and fans events out to them.
type hub struct {
	mu       sync.RWMutex
	clients  map[*client]struct{}
	commands map[string]CommandHandler
	closed   bool
	wg       sync.WaitGroup
}

func newHub() *hub {
	h := &hub{
		clients:  make(map[*client]struct{}),
		commands: make(map[string]CommandHandler),
	}
	// The heartbeat is built in so every client can rely on it existing.
	h.handle(FramePing, func(context.Context, json.RawMessage) (any, error) {
		return nil, nil
	})
	return h
}

// handle registers a command handler, replacing any previous one.
func (h *hub) handle(frameType string, handler CommandHandler) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.commands[frameType] = handler
}

// clientCount reports how many sockets are currently attached.
func (h *hub) clientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// add attaches a client and reports how many are now connected, counting this
// one. Returns false when the hub is already shutting down.
func (h *hub) add(c *client) (int, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed {
		return 0, false
	}
	h.clients[c] = struct{}{}
	// The pumps are counted here, under the same lock that shutdown takes, so
	// a concurrent shutdown either sees this client and waits for its
	// goroutines or refuses it outright — never returns while they start up.
	h.wg.Add(2)
	return len(h.clients), true
}

// remove detaches a client and closes its send channel exactly once, which is
// what tells its write pump to finish.
func (h *hub) remove(c *client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.clients[c]; !ok {
		return
	}
	delete(h.clients, c)
	close(c.send)
}

// broadcast fans a frame out to every live client, reporting problems through
// the server log.
func (h *hub) broadcast(frameType string, payload any) {
	h.fanout(frameType, payload, false)
}

// fanout is broadcast with a choice about logging. quiet exists for log frames:
// a warning raised while shipping a log line would itself be a log line, and
// the stream would feed on itself.
//
// A client whose buffer is full is dropped rather than waited on: the publisher
// is a background worker and must not block on a stalled browser.
func (h *hub) fanout(frameType string, payload any, quiet bool) {
	// Nothing is listening: skip the marshal entirely. Log frames arrive at
	// whatever rate the server logs, so this is the common case.
	if h.clientCount() == 0 {
		return
	}

	message, err := json.Marshal(outbound{Type: frameType, Payload: payload})
	if err != nil {
		if !quiet {
			logrus.Warnf("序列化事件帧失败: %v", err)
		}
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	for c := range h.clients {
		select {
		case c.send <- message:
		default:
			if !quiet {
				logrus.Warnf("事件推送缓冲区已满，断开一个慢客户端")
			}
			delete(h.clients, c)
			close(c.send)
		}
	}
}

// serve upgrades one request and runs its pumps until either side hangs up.
// buildHello is called after the client is attached, so the greeting can report
// a connection count that includes this one.
func (h *hub) serve(w http.ResponseWriter, r *http.Request, buildHello func(clients int) any) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		// Upgrade has already written its own error response.
		logrus.Warnf("升级 WebSocket 连接失败: %v", err)
		return
	}

	c := &client{hub: h, conn: conn, send: make(chan []byte, sendBuffer)}
	clients, ok := h.add(c)
	if !ok {
		_ = conn.Close()
		return
	}

	if message, err := json.Marshal(outbound{Type: FrameHello, Payload: buildHello(clients)}); err == nil {
		select {
		case c.send <- message:
		default:
		}
	}

	// The wait group was already incremented inside add, under the lock.
	go c.writePump()
	go c.readPump()
}

// readPump consumes client frames and keeps the read deadline alive. It owns
// all reads on the connection.
func (c *client) readPump() {
	defer func() {
		c.hub.wg.Done()
		c.hub.remove(c)
		_ = c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err,
				websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				logrus.Debugf("事件日志 WebSocket 读取结束: %v", err)
			}
			return
		}
		c.dispatch(message)
	}
}

// dispatch routes one inbound frame to its registered handler and replies on
// the same socket.
func (c *client) dispatch(message []byte) {
	var frame inbound
	if err := json.Unmarshal(message, &frame); err != nil {
		c.reply(FrameError, "无法解析的消息格式")
		return
	}

	c.hub.mu.RLock()
	handler, ok := c.hub.commands[frame.Type]
	c.hub.mu.RUnlock()
	if !ok {
		c.reply(FrameError, "未知的消息类型: "+frame.Type)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), writeWait)
	defer cancel()
	result, err := handler(ctx, frame.Payload)
	if err != nil {
		c.reply(FrameError, err.Error())
		return
	}
	// A ping carries no result; answering it still has to say pong.
	if frame.Type == FramePing {
		c.reply(FramePong, result)
		return
	}
	if result != nil {
		c.reply(frame.Type, result)
	}
}

// reply queues a frame for this client only, dropping it if the client is
// already backed up.
func (c *client) reply(frameType string, payload any) {
	message, err := json.Marshal(outbound{Type: frameType, Payload: payload})
	if err != nil {
		return
	}
	select {
	case c.send <- message:
	default:
	}
}

// writePump owns all writes on the connection, including the keepalive pings.
func (c *client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		c.hub.wg.Done()
		ticker.Stop()
		_ = c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed this client; say goodbye properly so the
				// browser reconnects instead of reporting a broken socket.
				_ = c.conn.WriteMessage(websocket.CloseMessage,
					websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// shutdown closes every socket and waits for their pumps, so the process does
// not exit with goroutines still writing.
func (h *hub) shutdown() {
	h.mu.Lock()
	if h.closed {
		h.mu.Unlock()
		return
	}
	h.closed = true
	for c := range h.clients {
		delete(h.clients, c)
		close(c.send)
	}
	h.mu.Unlock()
	h.wg.Wait()
}
