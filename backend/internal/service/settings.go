package service

import (
	"context"
	"strconv"
	"strings"

	"shale/internal/domain"
)

const (
	MaxAppNameRunes       = 50
	MaxLegalTextRunes     = 20000
	MaxAuditRetentionDays = 3650
)

type SettingsService struct {
	settings domain.SettingsRepository
}

func NewSettingsService(settings domain.SettingsRepository) *SettingsService {
	return &SettingsService{settings: settings}
}

type SettingsView struct {
	AppName            string
	DefaultLanguage    string
	ImprintText        string
	PrivacyText        string
	AuditRetentionDays int
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
	imprint := s.getText(ctx, domain.SettingImprintText)
	privacy := s.getText(ctx, domain.SettingPrivacyText)
	retention := 0
	if v, err := s.settings.Get(ctx, domain.SettingAuditRetentionDays); err == nil {
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && n >= 0 && n <= MaxAuditRetentionDays {
			retention = n
		}
	}
	return SettingsView{
		AppName:            name,
		DefaultLanguage:    lang,
		ImprintText:        imprint,
		PrivacyText:        privacy,
		AuditRetentionDays: retention,
	}, nil
}

func (s *SettingsService) getText(ctx context.Context, key string) string {
	v, err := s.settings.Get(ctx, key)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(v)
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

func (s *SettingsService) SetImprintText(ctx context.Context, text string) error {
	return s.setText(ctx, domain.SettingImprintText, text)
}

func (s *SettingsService) SetPrivacyText(ctx context.Context, text string) error {
	return s.setText(ctx, domain.SettingPrivacyText, text)
}

func (s *SettingsService) setText(ctx context.Context, key, text string) error {
	trimmed := strings.TrimSpace(text)
	if runeLen(trimmed) > MaxLegalTextRunes {
		return domain.Errorf(domain.KindInvalid, "%s must be at most %d characters", key, MaxLegalTextRunes)
	}
	return s.settings.Set(ctx, key, trimmed)
}

func (s *SettingsService) SetAuditRetentionDays(ctx context.Context, days int) error {
	if days < 0 || days > MaxAuditRetentionDays {
		return domain.Errorf(domain.KindInvalid, "audit_retention_days must be between 0 and %d", MaxAuditRetentionDays)
	}
	return s.settings.Set(ctx, domain.SettingAuditRetentionDays, strconv.Itoa(days))
}

func validRetention(v string) bool {
	n, err := strconv.Atoi(strings.TrimSpace(v))
	return err == nil && n >= 0 && n <= MaxAuditRetentionDays
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
	if v, ok := values[domain.SettingAuditRetentionDays]; !ok || !validRetention(v) {
		values[domain.SettingAuditRetentionDays] = "0"
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
