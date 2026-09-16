package service

import (
	"strings"
	"testing"
	"time"

	"shale/internal/domain"
)

func fixedClock(t time.Time) Clock {
	return func() time.Time { return t }
}

func TestAuthServiceLoginAndValidate(t *testing.T) {
	now := time.Unix(1_800_000_000, 0).UTC()
	auth := NewAuthService("hunter2", fixedClock(now))

	value, expiry, err := auth.Login("hunter2")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if want := now.Add(12 * time.Hour); !expiry.Equal(want) {
		t.Errorf("expiry = %v, want %v", expiry, want)
	}
	if !auth.Validate(value) {
		t.Errorf("Validate(valid session) = false, want true")
	}
}

func TestAuthServiceLoginWrongPassword(t *testing.T) {
	now := time.Unix(1_800_000_000, 0).UTC()
	auth := NewAuthService("hunter2", fixedClock(now))
	_, _, err := auth.Login("hunter3")
	if err == nil {
		t.Fatal("Login() with wrong password should fail")
	}
	if k, ok := domain.KindOf(err); !ok || k != domain.KindUnauthorized {
		t.Errorf("error kind = %q (%v), want unauthorized", k, ok)
	}
}

func TestAuthServiceValidateTampered(t *testing.T) {
	now := time.Unix(1_800_000_000, 0).UTC()
	auth := NewAuthService("hunter2", fixedClock(now))
	value, _, err := auth.Login("hunter2")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	tests := []struct {
		name  string
		value string
	}{
		{name: "tampered expiry", value: modifyPrefix(value, 1)},
		{name: "tampered signature", value: modifySuffix(value)},
		{name: "garbage", value: "garbage"},
		{name: "empty", value: ""},
		{name: "wrong key", value: NewAuthService("different", fixedClock(now)).mustLogin(t, "different")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if auth.Validate(tt.value) {
				t.Errorf("Validate(%q) = true, want false", tt.value)
			}
		})
	}
}

func TestAuthServiceExpiry(t *testing.T) {
	now := time.Unix(1_800_000_000, 0).UTC()

	auth := NewAuthService("pw", fixedClock(now.Add(-13*time.Hour)))
	value, _, err := auth.Login("pw")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	validator := NewAuthService("pw", fixedClock(now))
	if validator.Validate(value) {
		t.Errorf("expired session validated")
	}
}

func modifyPrefix(s string, delta byte) string {
	parts := strings.SplitN(s, ".", 2)
	first := []byte(parts[0])
	first[0] += delta
	return string(first) + "." + parts[1]
}

func modifySuffix(s string) string {
	i := strings.LastIndexByte(s, '.')
	if i < 0 {
		return s
	}
	sig := []byte(s[i+1:])
	if sig[0] == 'a' {
		sig[0] = 'b'
	} else {
		sig[0] = 'a'
	}
	return s[:i+1] + string(sig)
}

func (a *AuthService) mustLogin(t *testing.T, password string) string {
	t.Helper()
	value, _, err := a.Login(password)
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	return value
}
