package middleware

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"
)

// extractToken must normalise both transports identically. The query string is
// not a second-class path: file downloads and the admin event WebSocket have no
// other way to authenticate, and the token the frontend stores carries the
// "Bearer " prefix.
func TestExtractTokenNormalisesBothSources(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cases := []struct {
		name   string
		header string
		query  string
		want   string
	}{
		{name: "header with prefix", header: "Bearer abc.def.ghi", want: "abc.def.ghi"},
		{name: "header without prefix", header: "abc.def.ghi", want: "abc.def.ghi"},
		{name: "query with prefix", query: "Bearer abc.def.ghi", want: "abc.def.ghi"},
		{name: "query without prefix", query: "abc.def.ghi", want: "abc.def.ghi"},
		{name: "header wins over query", header: "Bearer from.header", query: "from.query", want: "from.header"},
		{name: "neither", want: ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			target := "/"
			if tc.query != "" {
				// The stored token contains a space, so it has to arrive
				// encoded — exactly as the browser sends it.
				target = "/?token=" + url.QueryEscape(tc.query)
			}
			request := httptest.NewRequest(http.MethodGet, target, nil)
			if tc.header != "" {
				request.Header.Set("Authorization", tc.header)
			}

			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = request

			if got := extractToken(c); got != tc.want {
				t.Errorf("extractToken() = %q, want %q", got, tc.want)
			}
		})
	}
}
