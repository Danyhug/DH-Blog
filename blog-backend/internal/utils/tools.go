package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"golang.org/x/text/encoding/simplifiedchinese"
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

// geoQueryClient is shared by all IP-location lookups; a 3s timeout keeps one
// failed lookup bounded while the access-log goroutines wait on it.
var geoQueryClient = &http.Client{Timeout: 3 * time.Second}

// IPLocationResponse IP地理位置API响应结构（ip-api.com）
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

// PconlineResponse whois.pconline.com.cn 的响应结构，正文为 GBK 编码。
type PconlineResponse struct {
	IP   string `json:"ip"`
	Pro  string `json:"pro"`
	City string `json:"city"`
	Addr string `json:"addr"`
	Err  string `json:"err"`
}

// GetIPLocation 查询IP所在城市。部署在国内时 ip-api.com 经常不可达，因此先查
// 国内可达的太平洋IP库（whois.pconline.com.cn），失败再回退到 ip-api.com。
func GetIPLocation(ip string) (string, error) {
	// 如果是本地IP或者内网IP，直接返回
	if ip == "127.0.0.1" || ip == "localhost" || strings.HasPrefix(ip, "192.168.") || strings.HasPrefix(ip, "10.") {
		return "本地网络", nil
	}

	if city, err := queryPconline(ip); err == nil {
		return city, nil
	}
	return queryIPAPI(ip)
}

// queryPconline 查询太平洋IP库，返回 "运营商/省份/城市" 格式。
func queryPconline(ip string) (string, error) {
	url := fmt.Sprintf("https://whois.pconline.com.cn/ipJson.jsp?ip=%s&json=true", ip)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("构造太平洋IP库请求失败: %w", err)
	}
	// 该接口会识别 UA 且响应为 GBK 编码
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := geoQueryClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("请求太平洋IP库失败: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取太平洋IP库响应失败: %w", err)
	}
	utf8Body, err := simplifiedchinese.GBK.NewDecoder().Bytes(body)
	if err != nil {
		return "", fmt.Errorf("解码太平洋IP库响应失败: %w", err)
	}

	var loc PconlineResponse
	if err := json.Unmarshal(utf8Body, &loc); err != nil {
		return "", fmt.Errorf("解析太平洋IP库响应失败: %w", err)
	}
	// 国外 IP 该库不准确（pro/city 为空），视为失败交给 ip-api.com 兜底
	if loc.Err != "" || (loc.Pro == "" && loc.City == "") {
		return "", fmt.Errorf("太平洋IP库未返回有效位置: %s", loc.Err)
	}

	// addr 形如 "北京市 联通"，最后一段是运营商
	ispType := "其他"
	if parts := strings.Split(loc.Addr, " "); len(parts) > 0 {
		ispType = classifyISP(parts[len(parts)-1])
	}
	return buildLocation(ispType, loc.Pro, loc.City, ""), nil
}

// queryIPAPI 查询 ip-api.com 作为国外 IP 的兜底。
func queryIPAPI(ip string) (string, error) {
	url := fmt.Sprintf("http://ip-api.com/json/%s?lang=zh-CN", ip)

	resp, err := geoQueryClient.Get(url)
	if err != nil {
		return "", fmt.Errorf("请求IP地理位置API失败: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取API响应失败: %w", err)
	}

	var location IPLocationResponse
	if err := json.Unmarshal(body, &location); err != nil {
		return "", fmt.Errorf("解析API响应失败: %w", err)
	}

	if location.Status != "success" {
		return "", fmt.Errorf("IP地理位置API返回错误状态: %s", location.Status)
	}
	return buildLocation(classifyISP(location.ISP), location.RegionName, location.City, location.Country), nil
}

// classifyISP 把供应商名称归一成前端展示的运营商类型
func classifyISP(isp string) string {
	switch {
	case strings.Contains(isp, "China Unicom"), strings.Contains(isp, "联通"):
		return "中国联通"
	case strings.Contains(isp, "China Telecom"), strings.Contains(isp, "Chinanet"), strings.Contains(isp, "电信"):
		return "中国电信"
	case strings.Contains(isp, "China Mobile"), strings.Contains(isp, "移动"):
		return "中国移动"
	case strings.Contains(isp, "China"):
		return "中国网络"
	default:
		return "其他"
	}
}

// buildLocation 拼接 "运营商/省份/城市"，直辖市等省市相同时去重
func buildLocation(ispType, region, city, country string) string {
	result := ispType
	switch {
	case region != "" && city != "" && city != region:
		return result + "/" + region + "/" + city
	case region != "":
		return result + "/" + region
	case city != "":
		return result + "/" + city
	case country != "":
		return result + "/" + country
	}
	return result
}
