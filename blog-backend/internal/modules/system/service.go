package system

import (
	"context"
)

type service struct{ settings *settingRepository }

func newService(settings *settingRepository) *service { return &service{settings: settings} }

func (s *service) configByType(ctx context.Context, configType string) (Config, error) {
	settings, err := s.settings.byType(ctx, configType)
	if err != nil {
		return Config{}, err
	}
	values := make(map[string]string, len(settings))
	for _, setting := range settings {
		values[setting.SettingKey] = setting.SettingValue
	}
	return configFrom(values), nil
}

type aiConfigSource struct{ service *service }

func (s aiConfigSource) LoadAITaggingConfig(ctx context.Context) (endpoint, apiKey, model, prompt string, err error) {
	config, err := s.service.configByType(ctx, ConfigTypeAI)
	if err != nil {
		return "", "", "", "", err
	}
	prompt, err = s.service.settings.value(ctx, SettingKeyAIPromptGetTags)
	if err != nil {
		prompt = DefaultTagsPrompt
	}
	return config.AIAPIURL, config.AIAPIKey, config.AIModel, prompt, nil
}

// LoadAISummaryConfig 与标签生成共用 AI 服务参数，只是换用摘要提示词。
func (s aiConfigSource) LoadAISummaryConfig(ctx context.Context) (endpoint, apiKey, model, prompt string, err error) {
	config, err := s.service.configByType(ctx, ConfigTypeAI)
	if err != nil {
		return "", "", "", "", err
	}
	prompt, err = s.service.settings.value(ctx, SettingKeyAIPromptGetAbstract)
	if err != nil {
		prompt = DefaultAbstractPrompt
	}
	return config.AIAPIURL, config.AIAPIKey, config.AIModel, prompt, nil
}

type commentPolicy struct{ service *service }

// CommentsOpen 反映后台「开放评论」开关。
func (p commentPolicy) CommentsOpen(ctx context.Context) (bool, error) {
	config, err := p.service.configByType(ctx, ConfigTypeBlog)
	if err != nil {
		return false, err
	}
	return config.OpenComment, nil
}
