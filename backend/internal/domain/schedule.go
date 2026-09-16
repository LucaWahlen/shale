package domain

import "time"

type Schedule struct {
	ID          string
	Title       string
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type ScheduleWithCount struct {
	Schedule
	EventCount    int
	FirstStartsAt string
	LastStartsAt  string
}
