package domain

const (
	SettingDefaultLanguage = "default_language"
	SettingAppName         = "app_name"
	SettingImprintText     = "imprint_text"
	SettingPrivacyText     = "privacy_text"
	LanguageEN             = "en"
	LanguageDE             = "de"
	DefaultAppName         = "shale"
)

func ValidLanguage(l string) bool {
	return l == LanguageEN || l == LanguageDE
}
