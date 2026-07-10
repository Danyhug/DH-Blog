package middleware

import "testing"

func TestGetResourceType(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{name: "heartbeat", path: "/api/user/heart", want: "heartbeat"},
		{name: "heartbeat suffix", path: "/api/user/heartbeat", want: "heartbeat"},
		{name: "article detail", path: "/api/article/42", want: "article"},
		{name: "admin route", path: "/api/admin/log/stats", want: "admin"},
		{name: "single segment", path: "/api/tags", want: "tags"},
		{name: "root", path: "/", want: ""},
		{name: "non api", path: "/assets/app.js", want: ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := getResourceType(test.path); got != test.want {
				t.Fatalf("getResourceType(%q) = %q, want %q", test.path, got, test.want)
			}
		})
	}
}
