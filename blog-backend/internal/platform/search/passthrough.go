package search

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
)

// maxPassthroughBody caps what the gateway will read back from an upstream, so
// a misbehaving provider cannot exhaust memory.
const maxPassthroughBody = 8 << 20

// PassthroughRequest is a call forwarded to one provider in its own wire
// format. Path is always chosen by the gateway from a fixed set of routes and
// never taken from the caller: a caller-controlled path would turn the gateway
// into an open proxy for anything on the provider's host.
type PassthroughRequest struct {
	Method string
	Path   string
	Query  url.Values
	Body   []byte
}

// PassthroughResponse is the upstream answer, returned verbatim so an existing
// provider SDK keeps working against the gateway.
type PassthroughResponse struct {
	Status       int
	ContentType  string
	Body         []byte
	Credits      int
	CostMicroUSD int
	// Kind is empty when the upstream returned 2xx. Otherwise it classifies
	// the failure for the breaker and the quota bookkeeping, even though the
	// body is still handed back to the caller unchanged.
	Kind ErrorKind
}

// OK reports whether the upstream succeeded.
func (r PassthroughResponse) OK() bool { return r.Status >= 200 && r.Status < 300 }

// Forwarder is a provider that can also be called in its native format.
type Forwarder interface {
	Provider
	Forward(ctx context.Context, req PassthroughRequest) (PassthroughResponse, error)
}

// forward performs the upstream call shared by both adapters. decorate adds the
// provider's own authentication headers; observe, when set, gets the upstream
// response header so an adapter can harvest quota information that only travels
// there. Both run for every status, since a rejected call still says something
// about the credential.
func forward(ctx context.Context, client *http.Client, provider, endpoint string,
	req PassthroughRequest, decorate func(*http.Request), observe func(http.Header)) (PassthroughResponse, error) {

	target := endpoint
	if len(req.Query) > 0 {
		target += "?" + req.Query.Encode()
	}

	var body io.Reader
	if len(req.Body) > 0 {
		body = bytes.NewReader(req.Body)
	}
	httpReq, err := http.NewRequestWithContext(ctx, req.Method, target, body)
	if err != nil {
		return PassthroughResponse{}, newError(provider, KindBadRequest, 0, err.Error())
	}
	httpReq.Header.Set("Accept", "application/json")
	if len(req.Body) > 0 {
		httpReq.Header.Set("Content-Type", "application/json")
	}
	decorate(httpReq)

	httpResp, err := client.Do(httpReq)
	if err != nil {
		return PassthroughResponse{}, newError(provider, classifyTransport(err), 0, err.Error())
	}
	defer func() { _ = httpResp.Body.Close() }()
	if observe != nil {
		observe(httpResp.Header)
	}

	payload, err := io.ReadAll(io.LimitReader(httpResp.Body, maxPassthroughBody))
	if err != nil {
		return PassthroughResponse{}, newError(provider, KindUnavailable, httpResp.StatusCode, err.Error())
	}

	response := PassthroughResponse{
		Status:      httpResp.StatusCode,
		ContentType: httpResp.Header.Get("Content-Type"),
		Body:        payload,
	}
	if response.ContentType == "" {
		response.ContentType = "application/json"
	}
	return response, nil
}

// Forward calls Brave in its native format.
func (p *BraveProvider) Forward(ctx context.Context, req PassthroughRequest) (PassthroughResponse, error) {
	if p.apiKey == "" {
		return PassthroughResponse{}, newError(ProviderBrave, KindAuthFailed, 0, "未配置 Brave API Key")
	}

	response, err := forward(ctx, p.client, ProviderBrave, p.baseURL+req.Path, req, func(httpReq *http.Request) {
		httpReq.Header.Set("X-Subscription-Token", p.apiKey)
	}, p.observeQuota)
	if err != nil {
		return PassthroughResponse{}, err
	}

	if response.OK() {
		// Brave bills per request regardless of what came back.
		response.Credits = 1
	} else {
		response.Kind = classifyStatus(response.Status)
	}
	return response, nil
}

// Forward calls Tavily in its native format.
func (p *TavilyProvider) Forward(ctx context.Context, req PassthroughRequest) (PassthroughResponse, error) {
	if p.apiKey == "" {
		return PassthroughResponse{}, newError(ProviderTavily, KindAuthFailed, 0, "未配置 Tavily API Key")
	}

	response, err := forward(ctx, p.client, ProviderTavily, p.baseURL+req.Path, req, func(httpReq *http.Request) {
		httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	}, nil)
	if err != nil {
		return PassthroughResponse{}, err
	}

	if response.OK() {
		response.Credits = p.creditsFrom(response.Body)
	} else {
		response.Kind = tavilyErrorKind(response.Status)
	}
	return response, nil
}

// creditsFrom reads the credit count Tavily reports, falling back to the
// documented price of the configured depth when the caller did not ask for
// usage details.
func (p *TavilyProvider) creditsFrom(body []byte) int {
	var payload struct {
		Usage struct {
			Credits int `json:"credits"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(body, &payload); err == nil && payload.Usage.Credits > 0 {
		return payload.Usage.Credits
	}
	if p.searchDepth == "advanced" {
		return 2
	}
	return 1
}

var _ Forwarder = (*BraveProvider)(nil)
var _ Forwarder = (*TavilyProvider)(nil)
var _ UsageReporter = (*TavilyProvider)(nil)
