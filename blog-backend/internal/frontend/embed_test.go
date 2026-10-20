package frontend

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"github.com/gin-gonic/gin"
)

func TestAcceptsGzip(t *testing.T) {
	cases := map[string]bool{
		"":                  false,
		"gzip":              true,
		"gzip, deflate, br": true,
		"br, GZIP;q=0.8":    true,
		"gzip;q=0":          false,
		"gzip; q=0.000":     false,
		"identity":          false,
		"*":                 true,
		"*;q=0":             false,
		"*, gzip;q=0":       false,
		"gzip;q=0, *":       false,
		"deflate, *;q=0.5":  true,
	}
	for header, want := range cases {
		if got := acceptsGzip(header); got != want {
			t.Errorf("acceptsGzip(%q) = %v, want %v", header, got, want)
		}
	}
}

func TestServeAssetsPrefersPrecompressed(t *testing.T) {
	gin.SetMode(gin.TestMode)
	assets := fstest.MapFS{
		"app.js":    {Data: []byte("plain-js")},
		"app.js.gz": {Data: []byte("gzipped-js")},
		"logo.png":  {Data: []byte("png")},
	}
	router := gin.New()
	router.GET("/assets/*filepath", serveAssets(assets))

	get := func(path, acceptEncoding string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		if acceptEncoding != "" {
			req.Header.Set("Accept-Encoding", acceptEncoding)
		}
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}

	rec := get("/assets/app.js", "gzip, br")
	if rec.Code != http.StatusOK || rec.Body.String() != "gzipped-js" {
		t.Fatalf("gzip client: got %d %q", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("Content-Encoding") != "gzip" || rec.Header().Get("Vary") != "Accept-Encoding" {
		t.Fatalf("gzip client headers: %v", rec.Header())
	}
	if ct := rec.Header().Get("Content-Type"); ct == "" || ct == "application/octet-stream" {
		t.Fatalf("gzip client should get the original type, got %q", ct)
	}

	rec = get("/assets/app.js", "")
	if rec.Body.String() != "plain-js" || rec.Header().Get("Content-Encoding") != "" {
		t.Fatalf("plain client: got %q with encoding %q", rec.Body.String(), rec.Header().Get("Content-Encoding"))
	}
	if rec.Header().Get("Vary") != "Accept-Encoding" {
		t.Fatalf("plain client should still see Vary, got %v", rec.Header())
	}

	rec = get("/assets/logo.png", "gzip")
	if rec.Body.String() != "png" || rec.Header().Get("Content-Encoding") != "" {
		t.Fatalf("file without .gz: got %q with encoding %q", rec.Body.String(), rec.Header().Get("Content-Encoding"))
	}

	if rec = get("/assets/", "gzip"); rec.Code != http.StatusNotFound {
		t.Fatalf("directory listing should be 404, got %d", rec.Code)
	}
	if rec = get("/assets/missing.js", "gzip"); rec.Code != http.StatusNotFound {
		t.Fatalf("missing file should be 404, got %d", rec.Code)
	}
}

// 桌宠素材走 /pet/ 而不是 /assets/：文件名没有内容哈希，浏览器得能直接把它当 ES module 加载。
func TestServePetPack(t *testing.T) {
	gin.SetMode(gin.TestMode)
	pack := fstest.MapFS{
		"kit/body.js":        {Data: []byte("export const a = 1;")},
		"whale/model.json":   {Data: []byte(`{"id":"whale"}`)},
		"whale/tex/face.png": {Data: []byte("png")},
	}
	router := gin.New()
	registerPetRoutes(router, pack)

	get := func(path string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		return rec
	}

	rec := get("/pet/kit/body.js")
	if rec.Code != http.StatusOK {
		t.Fatalf("/pet/kit/body.js: %d", rec.Code)
	}
	// 浏览器只接受 text/javascript 或 application/javascript 的模块，别的类型会拒绝执行
	if ct := rec.Header().Get("Content-Type"); ct != "text/javascript; charset=utf-8" && ct != "application/javascript" {
		t.Fatalf("module content type: %q", ct)
	}
	if cc := rec.Header().Get("Cache-Control"); cc != "public, max-age=600" {
		t.Fatalf("cache control: %q", cc)
	}

	if ct := get("/pet/whale/model.json").Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("json content type: %q", ct)
	}
	if ct := get("/pet/whale/tex/face.png").Header().Get("Content-Type"); ct != "image/png" {
		t.Fatalf("png content type: %q", ct)
	}

	// 缺文件要真 404，不能被 NoRoute 兜成 index.html
	if rec = get("/pet/whale/missing.png"); rec.Code != http.StatusNotFound {
		t.Fatalf("missing pet file should be 404, got %d", rec.Code)
	}
}
