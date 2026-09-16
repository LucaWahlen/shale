package domain

import "time"

type Event struct {
	ID          string
	ScheduleID  string
	Name        string
	Description string
	Location    string
	StartsAt    string
	AllDay      bool
	EndsAt      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
