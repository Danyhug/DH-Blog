package files

import "testing"

func TestResolveDownloadHeaders(t *testing.T) {
	tests := []struct {
		name            string
		mimeType        string
		preview         bool
		wantContentType string
		wantDisposition string
	}{
		{"非预览一律 attachment", "image/png", false, "image/png", "attachment"},
		{"空类型回退 octet-stream", "", false, "application/octet-stream", "attachment"},
		{"图片可 inline", "image/jpeg", true, "image/jpeg", "inline"},
		{"PDF 可 inline", "application/pdf", true, "application/pdf", "inline"},
		{"视频可 inline", "video/mp4", true, "video/mp4", "inline"},
		{"音频可 inline", "audio/mpeg", true, "audio/mpeg", "inline"},
		{"纯文本可 inline", "text/plain; charset=utf-8", true, "text/plain; charset=utf-8", "inline"},
		{"HTML 不可 inline", "text/html", true, "text/html", "attachment"},
		{"SVG 不可 inline", "image/svg+xml", true, "image/svg+xml", "attachment"},
		{"未知类型不可 inline", "application/octet-stream", true, "application/octet-stream", "attachment"},
		{"大小写不敏感", "IMAGE/PNG", true, "IMAGE/PNG", "inline"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			contentType, disposition := ResolveDownloadHeaders(tt.mimeType, tt.preview)
			if contentType != tt.wantContentType {
				t.Errorf("contentType = %q, want %q", contentType, tt.wantContentType)
			}
			if disposition != tt.wantDisposition {
				t.Errorf("disposition = %q, want %q", disposition, tt.wantDisposition)
			}
		})
	}
}
