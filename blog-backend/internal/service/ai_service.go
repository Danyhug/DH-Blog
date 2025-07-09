package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"dh-blog/internal/model"
	"dh-blog/internal/repository"
	"github.com/sirupsen/logrus"
)

type AIService interface {
	// GenerateTags 生成文章标签总结
	GenerateTags(text string) ([]string, error)
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
	aiConfig    *model.SystemConfig
	httpClient  *http.Client // 新增：HTTP 客户端
}

// NewAIService 创建新的AI服务实例
func NewAIService(settingRepo repository.SystemSettingRepository) AIService {
	// 创建带有超时的HTTP客户端
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// 从设置中加载AI配置
	settings, err := settingRepo.GetAllSettings()
	if err != nil {
		logrus.Errorf("加载AI配置失败: %v", err)
	}

	// 将设置列表转换为map
	settingsMap := make(map[string]string)
	for _, s := range settings {
		settingsMap[s.SettingKey] = s.SettingValue
	}

	// 创建配置对象
	config := model.FromSettingsMap(settingsMap)

	return &OpenAIService{
		settingRepo: settingRepo,
		aiConfig:    config,
		httpClient:  client,
	}
}

func (s *OpenAIService) Request(text string) (response OpenAIResponse, err error) {
	request := OpenAIRequest{
		Model: s.aiConfig.AiModel,
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

	newRequest, err := http.NewRequest(http.MethodPost, s.aiConfig.AiApiURL, bytes.NewBuffer(requestBody))
	if err != nil || newRequest == nil {
		logrus.Error("http请求创建失败", err)
		return
	}
	newRequest.Header.Set("Content-Type", "application/json")
	newRequest.Header.Set("Authorization", "Bearer "+s.aiConfig.AiApiKey)

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

	fmt.Println("测试响应体", string(body))
	return response, json.Unmarshal(body, &response)
}

func (s *OpenAIService) GenerateTags(text string) (result []string, err error) {
	// 使用从数据库加载的AI提示词
	prompt := s.aiConfig.AiPrompt
	if prompt == "" {
		logrus.Warn("AI提示词为空，使用默认提示词")
		// 如果数据库中没有配置提示词，可以使用一个默认值
		prompt = "请根据以下文章内容，提取出3-5个关键词作为文章标签，用逗号分隔。文章内容：{{.ArticleContent}}"
	}

	// 替换提示词中的占位符
	fullPrompt := fmt.Sprintf(prompt, text) // 假设只有一个占位符，且是文章内容

	response, err := s.Request(fullPrompt)
	if err != nil {
		logrus.Errorf("请求OpenAI API失败: %v", err)
		return
	}

	// 检查 Choices 是否为空
	if len(response.Choices) == 0 {
		return nil, fmt.Errorf("AI API 响应中没有 Choices，可能存在错误或无内容")
	}

	fmt.Println("测试响应数据", response)
	return []string{response.Choices[0].Message.Content}, nil
}
