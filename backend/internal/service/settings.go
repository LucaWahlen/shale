package service

import (
	"context"
	"strings"

	"shale/internal/domain"
)

const MaxAppNameRunes = 50

type SettingsService struct {
	settings domain.SettingsRepository
}

func NewSettingsService(settings domain.SettingsRepository) *SettingsService {
	return &SettingsService{settings: settings}
}

type SettingsView struct {
	AppName         string
	DefaultLanguage string
}

func (s *SettingsService) Get(ctx context.Context) (SettingsView, error) {
	lang := domain.LanguageEN
	v, err := s.settings.Get(ctx, domain.SettingDefaultLanguage)
	if err == nil && domain.ValidLanguage(v) {
		lang = v
	}
	name := domain.DefaultAppName
	v, err = s.settings.Get(ctx, domain.SettingAppName)
	if err == nil && ValidAppName(v) {
		name = v
	}
	return SettingsView{AppName: name, DefaultLanguage: lang}, nil
}

func (s *SettingsService) SetDefaultLanguage(ctx context.Context, lang string) error {
	if !domain.ValidLanguage(lang) {
		return domain.NewError(domain.KindInvalid, "default_language must be 'en' or 'de'")
	}
	return s.settings.Set(ctx, domain.SettingDefaultLanguage, lang)
}

func (s *SettingsService) SetAppName(ctx context.Context, name string) error {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return domain.NewError(domain.KindInvalid, "app_name must not be empty")
	}
	if runeLen(trimmed) > MaxAppNameRunes {
		return domain.Errorf(domain.KindInvalid, "app_name must be at most %d characters", MaxAppNameRunes)
	}
	return s.settings.Set(ctx, domain.SettingAppName, trimmed)
}

func ValidAppName(name string) bool {
	trimmed := strings.TrimSpace(name)
	return trimmed != "" && runeLen(trimmed) <= MaxAppNameRunes
}

func (s *SettingsService) All(ctx context.Context) (map[string]string, error) {
	values, err := s.settings.All(ctx)
	if err != nil {
		return nil, err
	}
	if v, ok := values[domain.SettingDefaultLanguage]; !ok || !domain.ValidLanguage(v) {
		values[domain.SettingDefaultLanguage] = domain.LanguageEN
	}
	if v, ok := values[domain.SettingAppName]; !ok || !ValidAppName(v) {
		values[domain.SettingAppName] = domain.DefaultAppName
	}
	return values, nil
}

func (s *SettingsService) ReplaceAll(ctx context.Context, values map[string]string) error {
	for k, v := range values {
		if err := s.settings.Set(ctx, k, v); err != nil {
			return err
		}
	}
	return nil
}
