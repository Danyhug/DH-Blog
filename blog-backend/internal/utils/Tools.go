package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// SendResponseData 类似于 Java 中的 SendResponseData 结构
type SendResponseData struct {
	Response http.ResponseWriter
	Message  string
}

// CsdnResponse 对应 Java 中的 Csdn 类
type CsdnResponse struct {
	Code int      `json:"code"`
	Msg  string   `json:"msg"`
	Data CsdnData `json:"data"`
}

type CsdnData struct {
	Address string `json:"address"`
	IP      string `json:"ip"`
}

// GetClientIP 获取客户端真实 IP 地址
func GetClientIP(r *http.Request) string {
	headers := []string{
		"X-Forwarded-For",
		"Proxy-Client-IP",
		"WL-Proxy-Client-IP",
		"HTTP_CLIENT_IP",
		"HTTP_X_FORWARDED_FOR",
	}

	for _, header := range headers {
		ip := r.Header.Get(header)
		if ip != "" && !strings.EqualFold(ip, "unknown") {
			// 对于 X-Forwarded-For，取第一个 IP
			if header == "X-Forwarded-For" {
				if ips := strings.Split(ip, ","); len(ips) > 0 {
					ip = strings.TrimSpace(ips[0])
				}
			}
			return ip
		}
	}

	ip := r.RemoteAddr
	// 去除端口部分
	if colon := strings.LastIndex(ip, ":"); colon != -1 {
		ip = ip[:colon]
	}

	// 将 IPv6 的本地回环地址转换为 IPv4 的本地回环地址
	if ip == "0:0:0:0:0:0:0:1" || ip == "::1" {
		ip = "127.0.0.1"
	}

	return ip
}

// ParseUserAgent 根据 User-Agent 获取操作系统和浏览器信息
func ParseUserAgent(userAgentString string) (string, error) {
	if userAgentString == "" {
		return "", errors.New("异常访问行为！")
	}

	var os, browser string

	// 解析操作系统
	switch {
	case strings.Contains(userAgentString, "Windows NT 10.0"):
		os = "Windows 10"
	case strings.Contains(userAgentString, "Windows NT 6.3"):
		os = "Windows 8.1"
	case strings.Contains(userAgentString, "Windows NT 6.2"):
		os = "Windows 8"
	case strings.Contains(userAgentString, "Windows NT 6.1"):
		os = "Windows 7"
	case strings.Contains(userAgentString, "Macintosh") || strings.Contains(userAgentString, "Mac OS X"):
		os = "Mac OS"
	case strings.Contains(userAgentString, "Android"):
		os = "Android"
	case strings.Contains(userAgentString, "iPhone") || strings.Contains(userAgentString, "iPad") || strings.Contains(userAgentString, "iPod"):
		os = "iOS"
	case strings.Contains(userAgentString, "HarmonyOS"):
		os = "鸿蒙"
	case strings.Contains(userAgentString, "Linux"):
		os = "Linux"
	}

	// 解析浏览器
	switch {
	case strings.Contains(userAgentString, "Edg/"):
		parts := strings.Split(userAgentString, "Edg/")
		if len(parts) > 1 {
			browser = "Edge " + strings.Split(parts[1], " ")[0]
		}
	case strings.Contains(userAgentString, "Chrome/"):
		parts := strings.Split(userAgentString, "Chrome/")
		if len(parts) > 1 {
			browser = "Chrome " + strings.Split(parts[1], " ")[0]
		}
	case strings.Contains(userAgentString, "Firefox/"):
		parts := strings.Split(userAgentString, "Firefox/")
		if len(parts) > 1 {
			browser = "Firefox " + strings.Split(parts[1], " ")[0]
		}
	case strings.Contains(userAgentString, "Safari/"):
		if strings.Contains(userAgentString, "Version/") {
			parts := strings.Split(userAgentString, "Version/")
			if len(parts) > 1 {
				browser = "Safari " + strings.Split(parts[1], " ")[0]
			}
		}
	}

	if os == "" && browser == "" {
		return "Unknown", nil
	}
	return fmt.Sprintf("%s; %s", os, browser), nil
}

// GetIpCity 获取 IP 地址对应的城市
func GetIpCity(ip string) (string, error) {
	url := "https://searchplugin.csdn.net/api/v1/ip/get?ip=" + ip

	// 创建请求
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	// 设置请求头
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 Safari/537.36")

	// 发送请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// 解析 JSON
	var csdnResp CsdnResponse
	if err := json.Unmarshal(body, &csdnResp); err != nil {
		return "", err
	}

	if csdnResp.Code == 200 {
		return csdnResp.Data.Address, nil
	}

	return "", fmt.Errorf("failed to get IP city, code: %d, msg: %s", csdnResp.Code, csdnResp.Msg)
}

// SendResponse 返回响应头数据
func SendResponse(data SendResponseData) {
	data.Response.WriteHeader(403)
	data.Response.Header().Set("Content-Type", "text/plain;charset=utf-8")
	if _, err := data.Response.Write([]byte("您的IP已被封禁，禁止访问")); err != nil {
		// 处理写入错误
		fmt.Printf("Failed to write response: %v\n", err)
	}
}
