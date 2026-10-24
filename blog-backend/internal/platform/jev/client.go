// Package jev is a client for TypeSafe AI's System One API, the endpoint that
// serves the Jev models. A System One model does not generate text: it reads a
// state and returns typed answers to the questions it was asked, each with a
// calibrated probability. That makes it a fit for decisions made inside a
// request path, where a chat model's latency and free-form output are not.
//
// Only the Choice question type is wrapped, because that is all the gateway
// asks. The package holds no business logic.
//
// API reference: https://docs.typesafe.ai/api/system-one
package jev

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	// DefaultBaseURL is the public API host.
	DefaultBaseURL = "https://api.typesafe.ai"
	// DefaultModel is the alias for the latest stable Jev release. Responses
	// name the versioned model that actually answered.
	DefaultModel = "jev-latest"

	// MaxChoices is the largest option set one Choice question accepts.
	MaxChoices = 255

	defaultTimeout = 10 * time.Second
	// questionID is the key the single question is sent and answered under.
	// The model never sees it.
	questionID = "pick"
)

// ErrNotConfigured means no API key has been set.
var ErrNotConfigured = errors.New("未配置 Jev API Key")

// Error is a failed API call. Status is zero when the request never got an
// HTTP response.
type Error struct {
	Status  int
	Message string
}

func (e *Error) Error() string {
	if e.Status > 0 {
		return fmt.Sprintf("jev: HTTP %d: %s", e.Status, e.Message)
	}
	return "jev: " + e.Message
}

// Choice is one Choice question. Criteria maps each option name to a
// description; both are shown to the model, so the descriptions should be
// written to tell the options apart.
type Choice struct {
	Instructions string
	Criteria     map[string]string
}

// Decision is the answer to a Choice question.
type Decision struct {
	// Model is the versioned id that answered, e.g. jev-1.14.0, even when an
	// alias was requested.
	Model string
	// Choice is the option with the highest probability.
	Choice string
	// Confidence is 0..1, derived from how peaked Probabilities is: a flat
	// distribution is low confidence even when one option is ahead.
	Confidence    float64
	Probabilities map[string]float64
	InputTokens   int
}

// Client calls POST /v1/systemone.
type Client struct {
	apiKey  string
	baseURL string
	model   string
	client  *http.Client
}

// New builds a client. An empty base URL or model selects the defaults.
func New(apiKey, baseURL, model string, client *http.Client) *Client {
	baseURL = strings.TrimSuffix(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	model = strings.TrimSpace(model)
	if model == "" {
		model = DefaultModel
	}
	if client == nil {
		client = &http.Client{Timeout: defaultTimeout}
	}
	return &Client{apiKey: strings.TrimSpace(apiKey), baseURL: baseURL, model: model, client: client}
}

// Configured reports whether the client has a key to call with.
func (c *Client) Configured() bool { return c != nil && c.apiKey != "" }

type question struct {
	Type         string            `json:"type"`
	Instructions string            `json:"instructions"`
	Criteria     map[string]string `json:"criteria"`
}

type request struct {
	Model     string              `json:"model"`
	State     any                 `json:"state"`
	Questions map[string]question `json:"questions"`
}

type response struct {
	Model   string `json:"model"`
	Answers map[string]struct {
		Type          string             `json:"type"`
		Choice        string             `json:"choice"`
		Confidence    float64            `json:"confidence"`
		Probabilities map[string]float64 `json:"probabilities"`
	} `json:"answers"`
	Usage struct {
		InputTokens int `json:"input_tokens"`
	} `json:"usage"`
}

// Choose asks one Choice question about state. State may be a string or any
// JSON-encodable value; structured fields help the model more than prose.
func (c *Client) Choose(ctx context.Context, state any, choice Choice) (Decision, error) {
	if !c.Configured() {
		return Decision{}, ErrNotConfigured
	}
	if len(choice.Criteria) == 0 {
		return Decision{}, &Error{Message: "选项为空"}
	}
	if len(choice.Criteria) > MaxChoices {
		return Decision{}, &Error{Message: fmt.Sprintf("选项数 %d 超过上限 %d", len(choice.Criteria), MaxChoices)}
	}

	body, err := json.Marshal(request{
		Model: c.model,
		State: state,
		Questions: map[string]question{
			questionID: {Type: "choice", Instructions: choice.Instructions, Criteria: choice.Criteria},
		},
	})
	if err != nil {
		return Decision{}, &Error{Message: err.Error()}
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/systemone", bytes.NewReader(body))
	if err != nil {
		return Decision{}, &Error{Message: err.Error()}
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	httpResp, err := c.client.Do(httpReq)
	if err != nil {
		return Decision{}, &Error{Message: err.Error()}
	}
	defer func() { _ = httpResp.Body.Close() }()

	raw, err := io.ReadAll(io.LimitReader(httpResp.Body, 1<<20))
	if err != nil {
		return Decision{}, &Error{Status: httpResp.StatusCode, Message: err.Error()}
	}
	if httpResp.StatusCode != http.StatusOK {
		return Decision{}, &Error{Status: httpResp.StatusCode, Message: errorMessage(httpResp.StatusCode, raw)}
	}

	var payload response
	if err := json.Unmarshal(raw, &payload); err != nil {
		return Decision{}, &Error{Status: httpResp.StatusCode, Message: "解析 Jev 响应失败: " + err.Error()}
	}
	answer, ok := payload.Answers[questionID]
	if !ok {
		return Decision{}, &Error{Status: httpResp.StatusCode, Message: "Jev 响应里没有该问题的答案"}
	}
	// The API guarantees the answer is one of the options; checking anyway
	// costs nothing and keeps a proxy or an API change from routing traffic to
	// a name the caller never offered.
	if _, offered := choice.Criteria[answer.Choice]; !offered {
		return Decision{}, &Error{Status: httpResp.StatusCode, Message: fmt.Sprintf("Jev 返回了未提供的选项 %q", answer.Choice)}
	}
	return Decision{
		Model:         payload.Model,
		Choice:        answer.Choice,
		Confidence:    answer.Confidence,
		Probabilities: answer.Probabilities,
		InputTokens:   payload.Usage.InputTokens,
	}, nil
}

// errorMessage pulls a readable reason out of an error body. The documented
// contract is only "a JSON body describing what went wrong", so the common
// shapes are tried and the raw body is the fallback. The region block (451)
// answers in RFC 7807 form, where the reason is in title.
func errorMessage(status int, body []byte) string {
	var shaped struct {
		Message string `json:"message"`
		Detail  any    `json:"detail"`
		Error   any    `json:"error"`
		Title   string `json:"title"`
	}
	if json.Unmarshal(body, &shaped) == nil {
		if text := firstText(shaped.Message, shaped.Detail, shaped.Error, shaped.Title); text != "" {
			if status == http.StatusUnavailableForLegalReasons {
				return text + "（TypeSafe 按出口 IP 限制地区，需让服务器经境外网络访问）"
			}
			return text
		}
	}
	if text := strings.TrimSpace(string(body)); text != "" {
		if runes := []rune(text); len(runes) > 300 {
			text = string(runes[:300])
		}
		return text
	}
	return http.StatusText(status)
}

func firstText(values ...any) string {
	for _, value := range values {
		switch typed := value.(type) {
		case string:
			if strings.TrimSpace(typed) != "" {
				return strings.TrimSpace(typed)
			}
		case map[string]any:
			if text, ok := typed["message"].(string); ok && strings.TrimSpace(text) != "" {
				return strings.TrimSpace(text)
			}
		case nil:
		default:
			if encoded, err := json.Marshal(typed); err == nil {
				return string(encoded)
			}
		}
	}
	return ""
}
