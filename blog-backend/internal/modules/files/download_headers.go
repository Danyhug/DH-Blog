package files

import (
	"mime"
	"strings"
)

// inlineSafeTypes 是允许以 inline 返回的确切 MIME 类型。
// 网盘里存的是用户上传的任意文件，若把 text/html、image/svg+xml 之类以
// inline + 原始 Content-Type 返回，浏览器会在本站同源下渲染它并执行其中的脚本，
// 等同于存储型 XSS。因此只放行浏览器能安全直接呈现的媒体类型。
var inlineSafeTypes = map[string]bool{
	"application/pdf": true,
	"text/plain":      true,
}

// inlineSafePrefixes 是允许 inline 的 MIME 前缀。
var inlineSafePrefixes = []string{"image/", "video/", "audio/"}

// inlineUnsafeTypes 是命中前缀但仍需拦截的例外。
// SVG 是可执行脚本的文档格式，通过 <img> 加载不受影响（<img> 会忽略
// Content-Disposition 且禁用脚本），但直接访问链接就会被当成页面执行。
var inlineUnsafeTypes = map[string]bool{
	"image/svg+xml": true,
	"image/svg":     true,
}

// ResolveDownloadHeaders 决定下载响应的 Content-Type 与 Content-Disposition。
// preview 为 true 时尽量返回 inline（供前端 <img>/<video>/<iframe> 直接渲染），
// 但仅限白名单类型；其余一律回退成 attachment，由浏览器下载而不是渲染。
func ResolveDownloadHeaders(mimeType string, preview bool) (contentType, disposition string) {
	contentType = strings.TrimSpace(mimeType)
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	if !preview {
		return contentType, "attachment"
	}

	// 存库的值可能带参数（如 "text/plain; charset=utf-8"），只按主类型判断
	base := contentType
	if parsed, _, err := mime.ParseMediaType(contentType); err == nil {
		base = parsed
	}
	base = strings.ToLower(base)

	if inlineUnsafeTypes[base] {
		return contentType, "attachment"
	}
	if inlineSafeTypes[base] {
		return contentType, "inline"
	}
	for _, prefix := range inlineSafePrefixes {
		if strings.HasPrefix(base, prefix) {
			return contentType, "inline"
		}
	}

	return contentType, "attachment"
}
