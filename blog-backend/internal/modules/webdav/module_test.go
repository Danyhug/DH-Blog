package webdav

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	usermodule "dh-blog/internal/modules/user"
	"dh-blog/internal/router"

	"github.com/gin-gonic/gin"
)

var _ UserAuthenticator = (*usermodule.Repository)(nil)

type stubUsers struct {
	username string
	password string
}

func (s stubUsers) Authenticate(username, password string) bool {
	return username == s.username && password == s.password
}

type stubFiles struct {
	path      string
	syncCalls int
}

func (s *stubFiles) GetStoragePath() string {
	return s.path
}

func (s *stubFiles) SyncFilesFromDiskDebounced() {
	s.syncCalls++
}

func TestRegisterRoutesRespectsEnabled(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		enabled    bool
		wantStatus int
	}{
		{name: "disabled", enabled: false, wantStatus: http.StatusNotFound},
		{name: "enabled", enabled: true, wantStatus: http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := gin.New()
			module := New(Dependencies{
				Enabled: tt.enabled,
				Prefix:  "/dav",
				Users:   stubUsers{},
				Files:   &stubFiles{path: t.TempDir()},
			})
			module.RegisterRoutes(&router.Routes{Engine: engine})

			request := httptest.NewRequest("PROPFIND", "/dav", nil)
			response := httptest.NewRecorder()
			engine.ServeHTTP(response, request)

			if response.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", response.Code, tt.wantStatus)
			}
		})
	}
}

func TestEnabledRegistersStandardAndWebDAVMethods(t *testing.T) {
	gin.SetMode(gin.TestMode)

	engine := gin.New()
	module := New(Dependencies{
		Enabled: true,
		Prefix:  "/dav",
		Users:   stubUsers{},
		Files:   &stubFiles{path: t.TempDir()},
	})
	module.RegisterRoutes(&router.Routes{Engine: engine})

	methods := []string{
		http.MethodGet,
		http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
		http.MethodHead,
		http.MethodOptions,
		http.MethodDelete,
		http.MethodConnect,
		http.MethodTrace,
		"PROPFIND",
		"PROPPATCH",
		"MKCOL",
		"COPY",
		"MOVE",
		"LOCK",
		"UNLOCK",
	}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			request := httptest.NewRequest(method, "/dav/file.txt", nil)
			response := httptest.NewRecorder()

			engine.ServeHTTP(response, request)

			if response.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want %d; method may not be registered", response.Code, http.StatusUnauthorized)
			}
		})
	}
}

func TestBasicAuthentication(t *testing.T) {
	gin.SetMode(gin.TestMode)

	engine := gin.New()
	module := New(Dependencies{
		Enabled: true,
		Prefix:  "/dav",
		Users:   stubUsers{username: "admin", password: "secret"},
		Files:   &stubFiles{path: t.TempDir()},
	})
	module.RegisterRoutes(&router.Routes{Engine: engine})

	tests := []struct {
		name       string
		username   string
		password   string
		setAuth    bool
		wantStatus int
	}{
		{name: "missing credentials", wantStatus: http.StatusUnauthorized},
		{name: "unknown user", username: "nobody", password: "secret", setAuth: true, wantStatus: http.StatusUnauthorized},
		{name: "wrong password", username: "admin", password: "wrong", setAuth: true, wantStatus: http.StatusUnauthorized},
		{name: "valid credentials", username: "admin", password: "secret", setAuth: true, wantStatus: http.StatusMultiStatus},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest("PROPFIND", "/dav", nil)
			if tt.setAuth {
				request.SetBasicAuth(tt.username, tt.password)
			}
			response := httptest.NewRecorder()

			engine.ServeHTTP(response, request)

			if response.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", response.Code, tt.wantStatus)
			}
			if tt.wantStatus == http.StatusUnauthorized && response.Header().Get("WWW-Authenticate") != basicAuthRealm {
				t.Fatalf("WWW-Authenticate = %q, want %q", response.Header().Get("WWW-Authenticate"), basicAuthRealm)
			}
		})
	}
}

func TestSuccessfulWriteTriggersDebouncedSync(t *testing.T) {
	gin.SetMode(gin.TestMode)

	storagePath := t.TempDir()
	files := &stubFiles{path: storagePath}
	engine := gin.New()
	module := New(Dependencies{
		Enabled: true,
		Prefix:  "/dav",
		Users:   stubUsers{username: "admin", password: "secret"},
		Files:   files,
	})
	module.RegisterRoutes(&router.Routes{Engine: engine})

	request := httptest.NewRequest(http.MethodPut, "/dav/note.txt", strings.NewReader("hello"))
	request.SetBasicAuth("admin", "secret")
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)

	if response.Code >= http.StatusBadRequest {
		t.Fatalf("PUT status = %d, want success", response.Code)
	}
	if files.syncCalls != 1 {
		t.Fatalf("sync calls = %d, want 1", files.syncCalls)
	}
}
