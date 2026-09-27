package domain

import (
	"crypto/rand"
	"encoding/hex"
)

func NewManageToken() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

const ManageTokenLength = 32

func ValidManageToken(token string) bool {
	if len(token) != ManageTokenLength {
		return false
	}
	_, err := hex.DecodeString(token)
	return err == nil
}
