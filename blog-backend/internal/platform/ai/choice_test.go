package ai

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

var threeOptions = ChoiceRequest{
	Instructions: "Pick one.",
	State:        map[string]any{"query": "x"},
	Options:      map[string]string{"brave": "web", "exa": "neural", "tavily": "agent"},
}

func chatServer(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *ChoiceClient) {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return server, NewChoiceClient(server.URL+"/v1", "sk-test", "test-model", server.Client())
}

func reply(w http.ResponseWriter, content string, logprobs any) {
	choice := map[string]any{"message": map[string]any{"content": content}}
	if logprobs != nil {
		choice["logprobs"] = map[string]any{"content": logprobs}
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"model": "test-model", "choices": []any{choice},
		"usage": map[string]int{"prompt_tokens": 99}})
}

func TestChooseAppendsPathAndReadsLogprobs(t *testing.T) {
	_, client := chatServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("path = %q, 只填基础地址时应补上 /chat/completions", r.URL.Path)
		}
		// 第一个 token 是空格，字母在第二个：概率要从字母那个 token 上取
		reply(w, " C", []any{
			map[string]any{"token": " ", "logprob": 0, "top_logprobs": []any{}},
			map[string]any{"token": "C", "logprob": math.Log(0.6), "top_logprobs": []any{
				map[string]any{"token": "C", "logprob": math.Log(0.6)},
				map[string]any{"token": " A", "logprob": math.Log(0.3)},
				map[string]any{"token": "hello", "logprob": math.Log(0.1)},
			}},
		})
	})

	result, err := client.Choose(context.Background(), threeOptions)
	if err != nil {
		t.Fatalf("Choose 返回错误: %v", err)
	}
	// 按名字排序编号：A = brave, B = exa, C = tavily；非选项 token 的概率不参与归一化
	if result.Choice != "tavily" || result.InputTokens != 99 {
		t.Fatalf("result = %+v", result)
	}
	if math.Abs(result.Probabilities["tavily"]-2.0/3) > 1e-9 || math.Abs(result.Probabilities["brave"]-1.0/3) > 1e-9 ||
		result.Probabilities["exa"] != 0 {
		t.Fatalf("probabilities = %v", result.Probabilities)
	}
}

func TestChooseRetriesWithoutLogprobsOnce(t *testing.T) {
	var calls atomic.Int32
	_, client := chatServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["logprobs"] == true {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":{"message":"logprobs is not supported for this model"}}`))
			return
		}
		reply(w, "Option B", nil)
	})

	for i := 0; i < 2; i++ {
		result, err := client.Choose(context.Background(), threeOptions)
		if err != nil {
			t.Fatalf("Choose 返回错误: %v", err)
		}
		if result.Choice != "exa" || result.Probabilities != nil {
			t.Fatalf("result = %+v", result)
		}
	}
	// 第一次 400 + 重试，第二次直接不带 logprobs
	if got := calls.Load(); got != 3 {
		t.Fatalf("请求次数 = %d, 期望 3", got)
	}
}

func TestChooseRejectsAnswerWithoutOption(t *testing.T) {
	_, client := chatServer(t, func(w http.ResponseWriter, r *http.Request) {
		reply(w, "I think Brave is best", nil)
	})
	if _, err := client.Choose(context.Background(), threeOptions); err == nil || !strings.Contains(err.Error(), "有效选项") {
		t.Fatalf("err = %v, 回答里没有独立的选项字母时应报错", err)
	}
}

func TestChooseExplainsUnusableReply(t *testing.T) {
	cases := []struct {
		name         string
		content      string
		reasoning    string
		finishReason string
		want         string
	}{
		{
			name: "推理模型只返回了思考过程", reasoning: "Weighing brave against tavily...",
			finishReason: "length", want: "思维强度",
		},
		{
			name: "输出预算被截断", finishReason: "length", want: "8 个输出 token",
		},
		{
			name: "答不到选项", content: "The best option is", finishReason: "length", want: "finish_reason=length",
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			_, client := chatServer(t, func(w http.ResponseWriter, r *http.Request) {
				_ = json.NewEncoder(w).Encode(map[string]any{
					"model": "test-model",
					"choices": []any{map[string]any{
						"message": map[string]any{
							"content":           testCase.content,
							"reasoning_content": testCase.reasoning,
						},
						"finish_reason": testCase.finishReason,
					}},
				})
			})
			_, err := client.Choose(context.Background(), threeOptions)
			if err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("err = %v, 期望包含 %q", err, testCase.want)
			}
			// 200 响应的语义错误不该伪装成 HTTP 请求失败
			if strings.Contains(err.Error(), "HTTP") {
				t.Fatalf("err = %v, 不应带 HTTP 状态前缀", err)
			}
		})
	}
}

func TestChooseSendsReasoningEffort(t *testing.T) {
	requestBody := make(chan map[string]any, 1)
	_, client := chatServer(t, func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		requestBody <- body
		reply(w, "A", nil)
	})
	client.SetReasoningEffort("none")

	if _, err := client.Choose(context.Background(), threeOptions); err != nil {
		t.Fatalf("Choose 返回错误: %v", err)
	}
	body := <-requestBody
	if body["reasoning_effort"] != "none" {
		t.Fatalf("request = %v, 思维强度应作为标准 reasoning_effort 字段发出", body)
	}
	if body["model"] != "test-model" || body["messages"] == nil {
		t.Fatalf("request = %v, 原有字段不该被覆盖掉", body)
	}
}

func TestChooseOmitsReasoningEffortWhenUnset(t *testing.T) {
	requestBody := make(chan map[string]any, 1)
	_, client := chatServer(t, func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		requestBody <- body
		reply(w, "A", nil)
	})

	if _, err := client.Choose(context.Background(), threeOptions); err != nil {
		t.Fatalf("Choose 返回错误: %v", err)
	}
	// 不选就完全不出现在请求体里，让模型用自己的默认档位
	if _, set := (<-requestBody)["reasoning_effort"]; set {
		t.Fatal("未设置思维强度时不该发送 reasoning_effort")
	}
}

func TestChooseReportsOpenAIError(t *testing.T) {
	_, client := chatServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"message":"Incorrect API key provided","type":"invalid_request_error"}}`))
	})
	_, err := client.Choose(context.Background(), threeOptions)
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Status != http.StatusUnauthorized || apiErr.Message != "Incorrect API key provided" {
		t.Fatalf("err = %v", err)
	}
}

func TestChoiceClientConfigured(t *testing.T) {
	if NewChoiceClient("", "k", "m", nil).Configured() || NewChoiceClient("http://x", "k", "", nil).Configured() {
		t.Error("缺地址或模型时不应视为已配置")
	}
	if !NewChoiceClient("http://localhost:11434/v1", "", "qwen2.5", nil).Configured() {
		t.Error("没有 Key 的本地服务也应可用")
	}
}
