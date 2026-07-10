package system

import (
	"context"
)

type service struct{ settings *settingRepository }

func newService(settings *settingRepository) *service { return &service{settings: settings} }

func (s *service) config(ctx context.Context) (Config, error) {
	settings, err := s.settings.all(ctx)
	if err != nil {
		return Config{}, err
	}
	values := make(map[string]string, len(settings))
	for _, setting := range settings {
		values[setting.SettingKey] = setting.SettingValue
	}
	return configFrom(values), nil
}

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

func (s *service) updateConfig(ctx context.Context, config Config) error {
	return s.settings.updateBatch(ctx, config.values(), "")
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
