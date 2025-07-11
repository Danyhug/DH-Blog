package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
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

// IPLocationResponse IP地理位置API响应结构
type IPLocationResponse struct {
	Status      string  `json:"status"`
	Country     string  `json:"country"`
	CountryCode string  `json:"countryCode"`
	Region      string  `json:"region"`
	RegionName  string  `json:"regionName"`
	City        string  `json:"city"`
	Zip         string  `json:"zip"`
	Lat         float64 `json:"lat"`
	Lon         float64 `json:"lon"`
	Timezone    string  `json:"timezone"`
	ISP         string  `json:"isp"`
	Org         string  `json:"org"`
	AS          string  `json:"as"`
	Query       string  `json:"query"`
}

// GetIPLocation 查询IP所在城市
func GetIPLocation(ip string) (string, error) {
	// 如果是本地IP或者内网IP，直接返回
	if ip == "127.0.0.1" || ip == "localhost" || strings.HasPrefix(ip, "192.168.") || strings.HasPrefix(ip, "10.") {
		return "本地网络", nil
	}

	// 创建HTTP客户端，设置超时时间
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	// 构建API请求URL
	url := fmt.Sprintf("http://ip-api.com/json/%s?lang=zh-CN", ip)

	// 发送HTTP请求
	resp, err := client.Get(url)
	if err != nil {
		return "", fmt.Errorf("请求IP地理位置API失败: %w", err)
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	// 读取响应内容
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取API响应失败: %w", err)
	}

	// 解析JSON响应
	var location IPLocationResponse
	if err := json.Unmarshal(body, &location); err != nil {
		return "", fmt.Errorf("解析API响应失败: %w", err)
	}

	// 检查API响应状态
	if location.Status != "success" {
		return "", fmt.Errorf("IP地理位置API返回错误状态: %s", location.Status)
	}

	// 解析运营商信息
	isp := location.ISP
	// 判断运营商类型
	ispType := "其他"
	if strings.Contains(isp, "China Unicom") || strings.Contains(isp, "联通") {
		ispType = "中国联通"
	} else if strings.Contains(isp, "China Telecom") || strings.Contains(isp, "电信") {
		ispType = "中国电信"
	} else if strings.Contains(isp, "China Mobile") || strings.Contains(isp, "移动") {
		ispType = "中国移动"
	} else if strings.Contains(isp, "China") {
		ispType = "中国网络"
	}

	// 构建符合前端格式的地理位置信息: "运营商/省份/城市" 或 "运营商/城市"
	result := ispType

	// 添加省份（如果有）
	if location.RegionName != "" {
		result += "/" + location.RegionName
	}

	// 添加城市（如果有）
	if location.City != "" {
		result += "/" + location.City
	} else if location.RegionName != "" {
		// 如果没有城市但有省份，则省份后不加斜杠
		return result, nil
	} else {
		// 如果既没有省份也没有城市，则加上国家名称
		result += "/" + location.Country
	}

	return result, nil
}
