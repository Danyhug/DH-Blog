package aigateway

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestGenerateAPIKeyShape(t *testing.T) {
	plain, err := GenerateAPIKey()
	if err != nil {
		t.Fatalf("GenerateAPIKey 返回错误: %v", err)
	}
	if len(plain) != len(apiKeyLiteral)+apiKeyRandomLength {
		t.Fatalf("Key 长度 = %d", len(plain))
	}
	if prefix := APIKeyPrefixOf(plain); prefix != plain[:len(apiKeyLiteral)+apiKeyPrefixDigits] {
		t.Fatalf("前缀 = %q", prefix)
	}

	other, err := GenerateAPIKey()
	if err != nil {
		t.Fatalf("GenerateAPIKey 返回错误: %v", err)
	}
	if plain == other {
		t.Fatal("两次生成的 Key 不应相同")
	}
	if HashAPIKey(plain) == HashAPIKey(other) {
		t.Fatal("不同 Key 的摘要不应相同")
	}
	if len(HashAPIKey(plain)) != 64 {
		t.Fatalf("摘要长度 = %d, 期望 sha256 的 64 位十六进制", len(HashAPIKey(plain)))
	}
}

func TestAuthenticateAcceptsValidKey(t *testing.T) {
	service, plain := newAuthTestService(t, func(*APIKey) {})

	key, err := service.authenticate(context.Background(), plain)
	if err != nil {
		t.Fatalf("authenticate 返回错误: %v", err)
	}
	if key.KeyPrefix != APIKeyPrefixOf(plain) {
		t.Fatalf("前缀 = %q", key.KeyPrefix)
	}
}

func TestAuthenticateRejections(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*APIKey)
		token   func(plain string) string
		wantErr error
	}{
		{
			name:    "缺少 Key",
			mutate:  func(*APIKey) {},
			token:   func(string) string { return "   " },
			wantErr: ErrMissingAPIKey,
		},
		{
			name:    "前缀不存在",
			mutate:  func(*APIKey) {},
			token:   func(string) string { return "gw_live_00000000deadbeef" },
			wantErr: ErrInvalidAPIKey,
		},
		{
			name:   "前缀命中但摘要不符",
			mutate: func(*APIKey) {},
			// Same prefix, different secret: the constant-time compare must fail.
			token:   func(plain string) string { return APIKeyPrefixOf(plain) + "tamperedtamperedtamperedtam" },
			wantErr: ErrInvalidAPIKey,
		},
		{
			name:    "已停用",
			mutate:  func(key *APIKey) { key.Enabled = false },
			token:   func(plain string) string { return plain },
			wantErr: ErrAPIKeyRevoked,
		},
		{
			name: "已过期",
			mutate: func(key *APIKey) {
				expired := time.Now().Add(-time.Hour)
				key.ExpireAt = &expired
			},
			token:   func(plain string) string { return plain },
			wantErr: ErrAPIKeyExpired,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service, plain := newAuthTestService(t, test.mutate)
			_, err := service.authenticate(context.Background(), test.token(plain))
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("错误 = %v, 期望 %v", err, test.wantErr)
			}
		})
	}
}

func TestAuthenticateSeesRevocationAfterInvalidation(t *testing.T) {
	service, plain := newAuthTestService(t, func(*APIKey) {})
	ctx := context.Background()

	key, err := service.authenticate(ctx, plain)
	if err != nil {
		t.Fatalf("authenticate 返回错误: %v", err)
	}

	if err := service.updateAPIKey(ctx, key.ID, map[string]any{"enabled": false}); err != nil {
		t.Fatalf("updateAPIKey 返回错误: %v", err)
	}
	if _, err := service.authenticate(ctx, plain); !errors.Is(err, ErrAPIKeyRevoked) {
		t.Fatalf("错误 = %v, 吊销后应立即生效", err)
	}
}

func TestAPIKeyAllows(t *testing.T) {
	tests := []struct {
		name     string
		allowed  string
		provider string
		want     bool
	}{
		{"空白名单放行全部", "", "brave", true},
		{"命中白名单", "brave,tavily", "tavily", true},
		{"不在白名单", "brave", "tavily", false},
		{"忽略空白项", " brave , ", "brave", true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			key := APIKey{AllowedProviders: test.allowed}
			if got := key.Allows(test.provider); got != test.want {
				t.Errorf("Allows(%q) = %v, 期望 %v", test.provider, got, test.want)
			}
		})
	}
}

func TestMinuteCountersEnforceLimit(t *testing.T) {
	counters := newMinuteCounters()
	now := time.Date(2026, 8, 1, 12, 30, 0, 0, time.UTC)

	for i := 0; i < 3; i++ {
		if !counters.allow(1, 3, now) {
			t.Fatalf("第 %d 次请求应通过", i+1)
		}
	}
	if counters.allow(1, 3, now) {
		t.Fatal("超出限额的请求应被拒绝")
	}
	// A different key has its own budget.
	if !counters.allow(2, 3, now) {
		t.Fatal("其他 Key 不应受影响")
	}
	// The window rolls over with the minute.
	if !counters.allow(1, 3, now.Add(time.Minute)) {
		t.Fatal("进入下一分钟后应重新放行")
	}
}

func TestMinuteCountersUnlimitedWhenLimitNonPositive(t *testing.T) {
	counters := newMinuteCounters()
	now := time.Now()
	for i := 0; i < 1000; i++ {
		if !counters.allow(1, 0, now) {
			t.Fatal("限额为 0 表示不限流")
		}
	}
}

func TestExtractAPIKey(t *testing.T) {
	tests := []struct {
		name          string
		authorization string
		apiKeyHeader  string
		want          string
	}{
		{"Bearer 前缀", "Bearer gw_live_abc", "", "gw_live_abc"},
		{"大小写不敏感", "bearer gw_live_abc", "", "gw_live_abc"},
		{"裸 token", "gw_live_abc", "", "gw_live_abc"},
		{"X-API-Key 优先", "Bearer other", "gw_live_abc", "gw_live_abc"},
		{"两者都缺失", "", "", ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := extractAPIKey(test.authorization, test.apiKeyHeader); got != test.want {
				t.Errorf("extractAPIKey = %q, 期望 %q", got, test.want)
			}
		})
	}
}

func TestMaskSecret(t *testing.T) {
	tests := []struct {
		secret string
		want   string
	}{
		{"", ""},
		{"short", "****"},
		{"tvly-dev-abcd1234", "tvly****1234"},
	}
	for _, test := range tests {
		t.Run(test.secret, func(t *testing.T) {
			if got := MaskSecret(test.secret); got != test.want {
				t.Errorf("MaskSecret(%q) = %q, 期望 %q", test.secret, got, test.want)
			}
		})
	}
}

// newAuthTestService builds a service with one API key and returns its
// plaintext. mutate adjusts the key before it is persisted.
func newAuthTestService(t *testing.T, mutate func(*APIKey)) (*Service, string) {
	t.Helper()
	module := newGatewayTestModule(t, gatewayTestConfig{})

	plain, err := GenerateAPIKey()
	if err != nil {
		t.Fatalf("GenerateAPIKey 返回错误: %v", err)
	}
	key := APIKey{
		Name:      "test",
		KeyPrefix: APIKeyPrefixOf(plain),
		KeyHash:   HashAPIKey(plain),
		Enabled:   true,
	}
	mutate(&key)
	if err := module.service.repo.createAPIKey(context.Background(), &key); err != nil {
		t.Fatalf("createAPIKey 返回错误: %v", err)
	}
	return module.service, plain
}
