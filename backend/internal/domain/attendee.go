package domain

import "time"

type Attendee struct {
	ID             string
	EventID        string
	Name           string
	NormalizedName string
	ManageToken    string
	CreatedAt      time.Time
}

const MaxNameRunes = 80
