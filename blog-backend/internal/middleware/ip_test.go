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

func TestSkipAccessLog(t *testing.T) {
	tests := []struct {
		name         string
		resourceType string
		urlPath      string
		rawQuery     string
		want         bool
	}{
		{name: "heartbeat skipped", resourceType: "heartbeat", urlPath: "/api/user/heart", want: true},
		{name: "gateway skipped", resourceType: "gateway", urlPath: "/api/gateway/chat", want: true},
		{name: "normal api logged", resourceType: "article", urlPath: "/api/article/list", want: false},
		{name: "admin api logged", resourceType: "admin", urlPath: "/api/admin/log/stats/visits", want: false},
		{name: "non api skipped", resourceType: "", urlPath: "/", want: true},
		{name: "static asset skipped", resourceType: "", urlPath: "/assets/app.js", want: true},
		{name: "favicon skipped", resourceType: "", urlPath: "/favicon.ico", want: true},
		{name: "phpunit probe in query skipped", resourceType: "v1", urlPath: "/api/v1/x", rawQuery: "p=/vendor/phpunit/phpunit/src/Util/PHP/eval-stdin.php", want: true},
		{name: "dotenv probe skipped", resourceType: "", urlPath: "/.env", want: true},
		{name: "docker api probe skipped", resourceType: "v1", urlPath: "/api/v1/proxy", rawQuery: "target=/containers/json", want: true},
		{name: "traversal in query skipped", resourceType: "article", urlPath: "/api/article/list", rawQuery: "lang=../../../../../../../../tmp/index1", want: true},
		{name: "encoded traversal skipped", resourceType: "article", urlPath: "/api/article/list", rawQuery: "path=..%2f..%2fetc%2fpasswd", want: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := skipAccessLog(test.resourceType, test.urlPath, test.rawQuery); got != test.want {
				t.Fatalf("skipAccessLog(%q, %q, %q) = %v, want %v",
					test.resourceType, test.urlPath, test.rawQuery, got, test.want)
			}
		})
	}
}
