package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"
)

// ChoiceClient asks an OpenAI-compatible chat model to pick one option from a
// closed set. The model answers with a single option letter rather than a
// name: a letter is one token, so the first token's logprobs are the model's
// whole distribution over the options, which is the only honest confidence a
// chat model can give — a number it writes about itself is not calibrated.
//
// Servers that do not support logprobs still work; the answer then comes back
// without probabilities and the caller decides what that means.
type ChoiceClient struct {
	endpoint string
	apiKey   string
	model    string
	client   *http.Client

	// noLogprobs remembers that this endpoint rejected the logprobs request
	// parameters, so only the first call pays for the retry.
	mu         sync.Mutex
	noLogprobs bool
}

// ChoiceRequest is one closed-set question.
type ChoiceRequest struct {
	// Instructions is the operator's prompt. The option list and the answer
	// format are appended by the client, because parsing depends on them.
	Instructions string
	// State is JSON-encoded into the user message.
	State any
	// Options maps each option name to its description.
	Options map[string]string
}

// ChoiceResult is the model's pick.
type ChoiceResult struct {
	Choice string
	// Probabilities over every option, normalised to sum to 1. Nil when the
	// server returned no usable logprobs.
	Probabilities map[string]float64
	Model         string
	InputTokens   int
}

// APIError is a non-200 answer from the chat endpoint.
type APIError struct {
	Status  int
	Message string
}

func (e *APIError) Error() string {
	if e.Status > 0 {
		return fmt.Sprintf("HTTP %d: %s", e.Status, e.Message)
	}
	return e.Message
}

// choiceLetters caps the option set: one letter per option keeps every answer
// a single token.
const choiceLetters = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"

// maxTopLogprobs is the most alternatives OpenAI returns per token.
const maxTopLogprobs = 20

const choiceAnswerFormat = "Reply with the letter of exactly one option and nothing else."

// NewChoiceClient builds a client. endpoint is the full chat completions URL,
// as in the blog's own AI settings; a bare base URL such as
// https://api.deepseek.com/v1 has /chat/completions appended.
func NewChoiceClient(endpoint, apiKey, model string, client *http.Client) *ChoiceClient {
	endpoint = strings.TrimSuffix(strings.TrimSpace(endpoint), "/")
	if endpoint != "" && !strings.HasSuffix(endpoint, "/chat/completions") {
		endpoint += "/chat/completions"
	}
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	return &ChoiceClient{
		endpoint: endpoint,
		apiKey:   strings.TrimSpace(apiKey),
		model:    strings.TrimSpace(model),
		client:   client,
	}
}

// Configured reports whether there is somewhere to send the question. A key
// is optional: local servers such as Ollama accept requests without one.
func (c *ChoiceClient) Configured() bool { return c != nil && c.endpoint != "" && c.model != "" }

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type choiceRequestBody struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature float64       `json:"temperature"`
	MaxTokens   int           `json:"max_tokens"`
	Logprobs    bool          `json:"logprobs,omitempty"`
	TopLogprobs int           `json:"top_logprobs,omitempty"`
}

type tokenLogprob struct {
	Token       string  `json:"token"`
	Logprob     float64 `json:"logprob"`
	TopLogprobs []struct {
		Token   string  `json:"token"`
		Logprob float64 `json:"logprob"`
	} `json:"top_logprobs"`
}

type choiceResponseBody struct {
	Model   string `json:"model"`
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		Logprobs *struct {
			Content []tokenLogprob `json:"content"`
		} `json:"logprobs"`
	} `json:"choices"`
	Usage struct {
		PromptTokens int `json:"prompt_tokens"`
	} `json:"usage"`
}

// Choose asks the question once.
func (c *ChoiceClient) Choose(ctx context.Context, req ChoiceRequest) (ChoiceResult, error) {
	if !c.Configured() {
		return ChoiceResult{}, &APIError{Message: "未配置接口地址或模型"}
	}
	if len(req.Options) == 0 {
		return ChoiceResult{}, &APIError{Message: "选项为空"}
	}
	if len(req.Options) > len(choiceLetters) {
		return ChoiceResult{}, &APIError{Message: fmt.Sprintf("选项数 %d 超过上限 %d", len(req.Options), len(choiceLetters))}
	}

	names := make([]string, 0, len(req.Options))
	for name := range req.Options {
		names = append(names, name)
	}
	sort.Strings(names)
	byLetter := make(map[string]string, len(names))
	var prompt strings.Builder
	prompt.WriteString(strings.TrimSpace(req.Instructions))
	prompt.WriteString("\n\nOptions:\n")
	for index, name := range names {
		letter := string(choiceLetters[index])
		byLetter[letter] = name
		fmt.Fprintf(&prompt, "%s. %s", letter, name)
		if description := strings.TrimSpace(req.Options[name]); description != "" {
			prompt.WriteString(": " + description)
		}
		prompt.WriteString("\n")
	}
	prompt.WriteString("\n" + choiceAnswerFormat)

	state, err := json.Marshal(req.State)
	if err != nil {
		return ChoiceResult{}, &APIError{Message: err.Error()}
	}
	body := choiceRequestBody{
		Model: c.model,
		Messages: []chatMessage{
			{Role: "system", Content: prompt.String()},
			{Role: "user", Content: string(state)},
		},
		Temperature: 0,
		// A letter is one token; the slack covers models that answer "A." or
		// "Option A" despite being told not to.
		MaxTokens: 8,
	}

	c.mu.Lock()
	withLogprobs := !c.noLogprobs
	c.mu.Unlock()
	if withLogprobs {
		body.Logprobs = true
		body.TopLogprobs = min(len(names), maxTopLogprobs)
	}

	payload, err := c.post(ctx, body)
	var apiErr *APIError
	if withLogprobs && errors.As(err, &apiErr) && rejectsLogprobs(apiErr) {
		c.mu.Lock()
		c.noLogprobs = true
		c.mu.Unlock()
		body.Logprobs, body.TopLogprobs = false, 0
		payload, err = c.post(ctx, body)
	}
	if err != nil {
		return ChoiceResult{}, err
	}
	if len(payload.Choices) == 0 {
		return ChoiceResult{}, &APIError{Status: http.StatusOK, Message: "模型响应里没有 choices"}
	}

	answer := payload.Choices[0]
	letter := answerLetter(answer.Message.Content, byLetter)
	if letter == "" {
		return ChoiceResult{}, &APIError{Status: http.StatusOK,
			Message: fmt.Sprintf("模型没有给出有效选项: %q", truncate(answer.Message.Content, 80))}
	}
	result := ChoiceResult{Choice: byLetter[letter], Model: payload.Model, InputTokens: payload.Usage.PromptTokens}
	if answer.Logprobs != nil {
		result.Probabilities = letterProbabilities(answer.Logprobs.Content, byLetter)
	}
	return result, nil
}

func (c *ChoiceClient) post(ctx context.Context, body choiceRequestBody) (choiceResponseBody, error) {
	encoded, err := json.Marshal(body)
	if err != nil {
		return choiceResponseBody{}, &APIError{Message: err.Error()}
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(encoded))
	if err != nil {
		return choiceResponseBody{}, &APIError{Message: err.Error()}
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	if c.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	httpResp, err := c.client.Do(httpReq)
	if err != nil {
		return choiceResponseBody{}, &APIError{Message: err.Error()}
	}
	defer func() { _ = httpResp.Body.Close() }()

	raw, err := io.ReadAll(io.LimitReader(httpResp.Body, 1<<20))
	if err != nil {
		return choiceResponseBody{}, &APIError{Status: httpResp.StatusCode, Message: err.Error()}
	}
	if httpResp.StatusCode != http.StatusOK {
		return choiceResponseBody{}, &APIError{Status: httpResp.StatusCode, Message: chatErrorMessage(httpResp.StatusCode, raw)}
	}
	var payload choiceResponseBody
	if err := json.Unmarshal(raw, &payload); err != nil {
		return choiceResponseBody{}, &APIError{Status: httpResp.StatusCode, Message: "解析模型响应失败: " + err.Error()}
	}
	return payload, nil
}

// rejectsLogprobs recognises a server refusing the logprobs parameters, as
// opposed to refusing the request for some other reason a retry cannot fix.
func rejectsLogprobs(err *APIError) bool {
	if err.Status != http.StatusBadRequest && err.Status != http.StatusUnprocessableEntity {
		return false
	}
	return strings.Contains(strings.ToLower(err.Message), "logprob")
}

// answerLetter finds the option letter in the model's reply: the whole reply
// when the model obeyed, otherwise the first standalone option letter.
func answerLetter(content string, byLetter map[string]string) string {
	content = strings.TrimSpace(content)
	if letter := strings.ToUpper(trimToken(content)); byLetter[letter] != "" {
		return letter
	}
	runes := []rune(content)
	for index, r := range runes {
		letter := string(unicode.ToUpper(r))
		if byLetter[letter] == "" {
			continue
		}
		before := index == 0 || !unicode.IsLetter(runes[index-1])
		after := index == len(runes)-1 || !unicode.IsLetter(runes[index+1])
		if before && after {
			return letter
		}
	}
	return ""
}

// letterProbabilities turns the logprobs of the answer token into a
// distribution over the options. The answer token is the first one that is an
// option letter, so a leading space or markdown token does not hide it.
func letterProbabilities(tokens []tokenLogprob, byLetter map[string]string) map[string]float64 {
	for _, token := range tokens {
		if byLetter[strings.ToUpper(trimToken(token.Token))] == "" {
			continue
		}
		mass := make(map[string]float64, len(byLetter))
		total := 0.0
		for _, alternative := range token.TopLogprobs {
			letter := strings.ToUpper(trimToken(alternative.Token))
			name := byLetter[letter]
			if name == "" {
				continue
			}
			probability := math.Exp(alternative.Logprob)
			mass[name] += probability
			total += probability
		}
		if total <= 0 {
			return nil
		}
		probabilities := make(map[string]float64, len(byLetter))
		for _, name := range byLetter {
			probabilities[name] = mass[name] / total
		}
		return probabilities
	}
	return nil
}

func trimToken(token string) string {
	return strings.TrimFunc(token, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) })
}

// chatErrorMessage reads the OpenAI error shape {"error":{"message":...}},
// falling back to the raw body.
func chatErrorMessage(status int, body []byte) string {
	var shaped struct {
		Error   json.RawMessage `json:"error"`
		Message string          `json:"message"`
	}
	if json.Unmarshal(body, &shaped) == nil {
		var nested struct {
			Message string `json:"message"`
		}
		if json.Unmarshal(shaped.Error, &nested) == nil && strings.TrimSpace(nested.Message) != "" {
			return strings.TrimSpace(nested.Message)
		}
		var flat string
		if json.Unmarshal(shaped.Error, &flat) == nil && strings.TrimSpace(flat) != "" {
			return strings.TrimSpace(flat)
		}
		if strings.TrimSpace(shaped.Message) != "" {
			return strings.TrimSpace(shaped.Message)
		}
	}
	if text := strings.TrimSpace(string(body)); text != "" {
		return truncate(text, 300)
	}
	return http.StatusText(status)
}

func truncate(text string, limit int) string {
	if runes := []rune(text); len(runes) > limit {
		return string(runes[:limit])
	}
	return text
}
