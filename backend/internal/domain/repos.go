package domain

import "context"

type ScheduleRepository interface {
	List(ctx context.Context) ([]ScheduleWithCount, error)
	ListPage(ctx context.Context, q ScheduleListQuery) (SchedulePage, error)
	GetByID(ctx context.Context, id string) (Schedule, error)
	Create(ctx context.Context, s *Schedule) error
	Update(ctx context.Context, s *Schedule) error
	Delete(ctx context.Context, id string) error
	DeleteAll(ctx context.Context) error

	Duplicate(ctx context.Context, id string) (string, error)
}

type EventRepository interface {
	ListBySchedule(ctx context.Context, scheduleID string) ([]Event, error)
	GetByID(ctx context.Context, id string) (Event, error)
	Create(ctx context.Context, e *Event) error
	Update(ctx context.Context, e *Event) error
	Delete(ctx context.Context, id string) error
}

type AttendeeRepository interface {
	ListByEvent(ctx context.Context, eventID string) ([]Attendee, error)
	GetByID(ctx context.Context, id string) (Attendee, error)
	FindByEventAndNormalizedName(ctx context.Context, eventID string, normalized string) (Attendee, error)
	Create(ctx context.Context, a *Attendee) error
	UpdateToken(ctx context.Context, id string, token string) error
	Delete(ctx context.Context, id string) error
}

type SettingsRepository interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key, value string) error
	All(ctx context.Context) (map[string]string, error)
}

type TransactionManager interface {
	Do(ctx context.Context, fn func(ctx context.Context) error) error
}
