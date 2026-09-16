package domain

import (
	"strings"
	"unicode/utf8"
)

func NormalizeName(name string) string {
	return strings.ToLower(strings.Join(strings.Fields(name), " "))
}

func ValidateName(name string) (string, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return "", NewError(KindInvalid, "name must not be empty")
	}
	if utf8.RuneCountInString(trimmed) > MaxNameRunes {
		return "", Errorf(KindInvalid, "name must be at most %d characters", MaxNameRunes)
	}
	return trimmed, nil
}
