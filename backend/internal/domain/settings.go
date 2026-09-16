package domain

const (
	SettingDefaultLanguage = "default_language"
	SettingAppName         = "app_name"
	LanguageEN             = "en"
	LanguageDE             = "de"
	DefaultAppName         = "shale"
)

func ValidLanguage(l string) bool {
	return l == LanguageEN || l == LanguageDE
}
