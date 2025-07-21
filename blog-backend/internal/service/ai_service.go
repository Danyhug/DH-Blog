package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"text/template"
	"time"

	"dh-blog/internal/database"
	"dh-blog/internal/dhcache"
	"dh-blog/internal/model"
	"dh-blog/internal/repository"

	"github.com/sirupsen/logrus"
)

type AIService interface {
	// GenerateTags 生成文章标签总结
	// existingTags 参数是系统中所有已存在的标签，用于AI参考
	GenerateTags(text string, existingTags []string) ([]string, error)
	// ClearConfigCache 清除AI配置缓存
	ClearConfigCache()
}

// OpenAIRequest 定义向OpenAI API发送的请求结构
type OpenAIRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

// Message 定义对话消息结构
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OpenAIResponse 定义从OpenAI API接收的响应结构
type OpenAIResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int      `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
}

// Choice 定义响应中的选择项结构
type Choice struct {
	Index   int     `json:"index"`
	Message Message `json:"message"`
}

type OpenAIService struct {
	settingRepo repository.SystemSettingRepository
	httpClient  *http.Client  // HTTP 客户端
	cache       dhcache.Cache // 缓存实例
}

const (
	// AiConfigCacheKey AI配置缓存键
	AiConfigCacheKey = "ai:config"
	// ExpireShort AI缓存过期时间
	ExpireShort = time.Minute * 5 // 短期缓存，5分钟
	ExpireLong  = time.Hour * 2   // 长期缓存，2小时
)

// NewAIService 创建新的AI服务实例
func NewAIService(settingRepo repository.SystemSettingRepository, cache dhcache.Cache) AIService {
	// 创建带有超时的HTTP客户端
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	return &OpenAIService{
		settingRepo: settingRepo,
		httpClient:  client,
		cache:       cache,
	}
}

// ClearConfigCache 清除AI配置缓存
func (s *OpenAIService) ClearConfigCache() {
	if deleted := s.cache.Delete(AiConfigCacheKey); !deleted {
		logrus.Warnf("清除AI配置缓存失败: 缓存中未找到")
	} else {
		logrus.Debug("AI配置缓存已清除")
	}
}

// getLatestConfig 获取最新的AI配置，优先从缓存获取
func (s *OpenAIService) getLatestConfig() (*model.SystemConfig, error) {
	// 尝试从缓存获取
	if cached, found := s.cache.Get(AiConfigCacheKey); found {
		if config, ok := cached.(*model.SystemConfig); ok {
			logrus.Debug("从缓存获取AI配置")
			return config, nil
		} else {
			logrus.Warn("AI配置缓存类型转换失败，将从数据库重新获取")
		}
	}

	// 从设置中加载AI配置
	settings, err := s.settingRepo.GetAllSettings()
	if err != nil {
		logrus.Errorf("加载AI配置失败: %v", err)
		return nil, err
	}

	// 将设置列表转换为map
	settingsMap := make(map[string]string)
	for _, s := range settings {
		settingsMap[s.SettingKey] = s.SettingValue
	}

	// 创建配置对象
	config := model.FromSettingsMap(settingsMap)

	// 存入缓存，设置5分钟过期
	_ = s.cache.Set(AiConfigCacheKey, config, ExpireShort)
	logrus.Debug("AI配置已缓存")

	return config, nil
}

func (s *OpenAIService) Request(text string) (response OpenAIResponse, err error) {
	// 获取最新配置
	config, err := s.getLatestConfig()
	if err != nil {
		return response, fmt.Errorf("获取AI配置失败: %w", err)
	}

	request := OpenAIRequest{
		Model: config.AiModel,
		Messages: []Message{
			{
				Role:    "user",
				Content: text,
			},
		},
	}

	requestBody, err := json.Marshal(request)
	if err != nil {
		return
	}

	newRequest, err := http.NewRequest(http.MethodPost, config.AiApiURL, bytes.NewBuffer(requestBody))
	if err != nil || newRequest == nil {
		logrus.Error("http请求创建失败", err)
		return
	}
	newRequest.Header.Set("Content-Type", "application/json")
	newRequest.Header.Set("Authorization", "Bearer "+config.AiApiKey)

	do, err := s.httpClient.Do(newRequest) // 使用 s.httpClient
	if err != nil {
		return
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			logrus.Error("响应体关闭失败", err)
		}
	}(do.Body)

	body, err := io.ReadAll(do.Body)
	if err != nil {
		return
	}

	logrus.Debug("AI响应体", string(body))
	return response, json.Unmarshal(body, &response)
}

// 为AI生成的标签创建缓存键
func generateTagsCacheKey(text string) string {
	// 使用文本的前50个字符作为键的一部分
	shortText := text
	if len(shortText) > 50 {
		shortText = shortText[:50]
	}
	return "ai:tags:" + shortText
}

func (s *OpenAIService) GenerateTags(text string, existingTags []string) (result []string, err error) {
	// 生成缓存键
	cacheKey := generateTagsCacheKey(text)

	// 尝试从缓存获取
	if cached, found := s.cache.Get(cacheKey); found {
		if tags, ok := cached.([]string); ok {
			logrus.Debug("从缓存获取AI生成的标签")
			return tags, nil
		} else {
			logrus.Warn("AI生成的标签缓存类型转换失败，将重新生成")
		}
	}

	// 使用从数据库加载的AI提示词
	prompt, err := s.settingRepo.GetSetting("ai_prompt_get_tags")
	if err != nil {
		logrus.Errorf("从数据库获取AI提示词失败: %v", err)
		// 如果数据库中没有配置提示词，可以使用一个默认值
		prompt = database.DefaultTagsPrompt
	}

	// 替换提示词中的占位符
	tmpl, err := template.New("prompt").Parse(prompt)
	if err != nil {
		logrus.Errorf("解析提示词模板失败: %v", err)
		return nil, err
	}

	// 准备数据
	data := struct {
		Article string
		Tags    string
	}{
		Article: text,
		Tags:    strings.Join(existingTags, ", "),
	}

	// 使用Buffer存储填充后的内容
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		logrus.Errorf("执行提示词模板失败: %v", err)
		return nil, err
	}

	logrus.Infof("AI提示词: %s", buf.String())
	response, err := s.Request(buf.String())
	if err != nil {
		logrus.Errorf("请求OpenAI API失败: %v", err)
		return nil, err
	}

	// 检查 Choices 是否为空
	if len(response.Choices) == 0 {
		return nil, fmt.Errorf("AI API 响应中没有 Choices，可能存在错误或无内容")
	}

	// 尝试解析AI返回的内容为JSON数组
	content := response.Choices[0].Message.Content
	// 清理内容，确保它是一个有效的JSON
	content = strings.TrimSpace(content)

	// 如果内容被反引号或其他格式包围，尝试提取JSON部分
	if strings.Contains(content, "[") && strings.Contains(content, "]") {
		start := strings.Index(content, "[")
		end := strings.LastIndex(content, "]") + 1
		if start >= 0 && end > start {
			content = content[start:end]
		}
	}

	err = json.Unmarshal([]byte(content), &result)
	if err != nil {
		// 如果JSON解析失败，尝试使用简单的逗号分隔方式
		logrus.Warnf("解析AI返回的JSON失败: %v，尝试使用逗号分隔方式", err)

		// 移除可能的引号和括号
		content = strings.ReplaceAll(content, "[", "")
		content = strings.ReplaceAll(content, "]", "")
		content = strings.ReplaceAll(content, "\"", "")
		content = strings.ReplaceAll(content, "'", "")

		// 按逗号分隔
		tags := strings.Split(content, ",")
		for _, tag := range tags {
			tag = strings.TrimSpace(tag)
			if tag != "" {
				result = append(result, tag)
			}
		}
	}

	// 清理标签，移除空白标签
	var cleanTags []string
	for _, tag := range result {
		tag = strings.TrimSpace(tag)
		if tag != "" {
			cleanTags = append(cleanTags, tag)
		}
	}

	logrus.Infof("AI生成的标签: %v", cleanTags)

	// 将结果存入缓存，设置1小时过期
	_ = s.cache.Set(cacheKey, cleanTags, ExpireLong)
	logrus.Debug("AI生成的标签已缓存")

	return cleanTags, nil
}
