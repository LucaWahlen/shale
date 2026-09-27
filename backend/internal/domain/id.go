package domain

import "github.com/google/uuid"

func NewID() string {
	return uuid.NewString()
}

func ValidID(id string) bool {
	_, err := uuid.Parse(id)
	return err == nil
}
