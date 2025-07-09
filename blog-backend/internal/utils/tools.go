package utils

import (
	"net/http"
	"strings"
)

// GetClientIP 获取客户端真实 IP 地址
func GetClientIP(r *http.Request) string {
	ip := r.Header.Get("X-Forwarded-For")
	if ip == "" || strings.EqualFold(ip, "unknown") {
		ip = r.Header.Get("Proxy-Client-IP")
	}
	if ip == "" || strings.EqualFold(ip, "unknown") {
		ip = r.Header.Get("WL-Proxy-Client-IP")
	}
	if ip == "" || strings.EqualFold(ip, "unknown") {
		ip = r.Header.Get("HTTP_CLIENT_IP")
	}
	if ip == "" || strings.EqualFold(ip, "unknown") {
		ip = r.Header.Get("HTTP_X_FORWARDED_FOR")
	}
	if ip == "" || strings.EqualFold(ip, "unknown") {
		ip = r.RemoteAddr
	}

	// 如果是 IPv6 的本地回环地址，转换为 IPv4
	if ip == "::1" || strings.HasPrefix(ip, "[::1]") {
		ip = "127.0.0.1"
	}

	// 如果是 "IP:Port" 格式，只取 IP 部分
	if strings.Contains(ip, ":") {
		parts := strings.Split(ip, ":")
		ip = parts[0]
	}

	return ip
}

// ParseUserAgent 根据 User-Agent 字符串解析操作系统和浏览器信息
// 这是一个简化的版本，可以根据需要扩展
func ParseUserAgent(userAgentString string) (os, browser string) {
	if userAgentString == "" {
		return "", ""
	}

	// 解析操作系统
	if strings.Contains(userAgentString, "Windows NT 10.0") {
		os = "Windows 10"
	} else if strings.Contains(userAgentString, "Windows NT 6.3") {
		os = "Windows 8.1"
	} else if strings.Contains(userAgentString, "Windows NT 6.2") {
		os = "Windows 8"
	} else if strings.Contains(userAgentString, "Windows NT 6.1") {
		os = "Windows 7"
	} else if strings.Contains(userAgentString, "Macintosh") || strings.Contains(userAgentString, "Mac OS X") {
		os = "Mac OS"
	} else if strings.Contains(userAgentString, "Android") {
		os = "Android"
	} else if strings.Contains(userAgentString, "iPhone") || strings.Contains(userAgentString, "iPad") || strings.Contains(userAgentString, "iPod") {
		os = "iOS"
	} else if strings.Contains(userAgentString, "HarmonyOS") {
		os = "鸿蒙"
	} else if strings.Contains(userAgentString, "Linux") {
		os = "Linux"
	} else {
		os = userAgentString
	}

	// 解析浏览器
	if strings.Contains(userAgentString, "Edg/") {
		browser = "Edge"
	} else if strings.Contains(userAgentString, "Chrome/") {
		browser = "Chrome"
	} else if strings.Contains(userAgentString, "Firefox/") {
		browser = "Firefox"
	} else if strings.Contains(userAgentString, "Safari/") {
		browser = "Safari"
	} else {
		browser = userAgentString
	}

	return os, browser
}
