package service

import (
	"context"
	"strings"

	"shale/internal/domain"
)

type EventService struct {
	schedules domain.ScheduleRepository
	events    domain.EventRepository
}

func NewEventService(schedules domain.ScheduleRepository, events domain.EventRepository) *EventService {
	return &EventService{schedules: schedules, events: events}
}

type EventInput struct {
	Name        string
	Description string
	Location    string
	StartsAt    string
	AllDay      bool
	EndsAt      string
}

type EventPatch struct {
	Name        *string
	Description *string
	Location    *string
	StartsAt    *string
	AllDay      *bool
	EndsAt      *string
}

const maxLocationRunes = 500

func validateLocation(loc string) (string, error) {
	trimmed := strings.TrimSpace(loc)
	if runeLen(trimmed) > maxLocationRunes {
		return "", domain.Errorf(domain.KindInvalid, "location must be at most %d characters", maxLocationRunes)
	}
	return trimmed, nil
}

func validateEventInput(in EventInput) (EventInput, error) {
	name, err := validateTitle(in.Name)
	if err != nil {
		return in, err
	}
	desc, err := validateDescription(in.Description)
	if err != nil {
		return in, err
	}
	loc, err := validateLocation(in.Location)
	if err != nil {
		return in, err
	}
	if !domain.ValidStartsAt(in.StartsAt) {
		return in, domain.NewError(domain.KindInvalid, "starts_at must be in YYYY-MM-DDTHH:MM form")
	}
	startsAt := in.StartsAt
	if in.AllDay {
		normalized, ok := domain.NormalizeAllDayStartsAt(startsAt)
		if !ok {
			return in, domain.NewError(domain.KindInvalid, "starts_at must be in YYYY-MM-DDTHH:MM form")
		}
		startsAt = normalized
		if in.EndsAt != "" {
			return in, domain.NewError(domain.KindInvalid, "all-day events must not have an end time")
		}
	} else if in.EndsAt != "" && !domain.ValidEndsAt(startsAt, in.EndsAt) {
		return in, domain.NewError(domain.KindInvalid, "ends_at must be in YYYY-MM-DDTHH:MM form and after starts_at")
	}
	return EventInput{Name: name, Description: desc, Location: loc, StartsAt: startsAt, AllDay: in.AllDay, EndsAt: in.EndsAt}, nil
}

func (s *EventService) Create(ctx context.Context, scheduleID string, in EventInput) (domain.Event, error) {
	if _, err := s.schedules.GetByID(ctx, scheduleID); err != nil {
		return domain.Event{}, err
	}
	clean, err := validateEventInput(in)
	if err != nil {
		return domain.Event{}, err
	}
	ev := domain.Event{
		ScheduleID:  scheduleID,
		Name:        clean.Name,
		Description: clean.Description,
		Location:    clean.Location,
		StartsAt:    clean.StartsAt,
		AllDay:      clean.AllDay,
		EndsAt:      clean.EndsAt,
	}
	if err := s.events.Create(ctx, &ev); err != nil {
		return domain.Event{}, err
	}
	return ev, nil
}

func (s *EventService) Patch(ctx context.Context, id string, patch EventPatch) (domain.Event, error) {
	existing, err := s.events.GetByID(ctx, id)
	if err != nil {
		return domain.Event{}, err
	}
	in := EventInput{
		Name:        existing.Name,
		Description: existing.Description,
		Location:    existing.Location,
		StartsAt:    existing.StartsAt,
		AllDay:      existing.AllDay,
		EndsAt:      existing.EndsAt,
	}
	if patch.Name != nil {
		in.Name = *patch.Name
	}
	if patch.Description != nil {
		in.Description = *patch.Description
	}
	if patch.Location != nil {
		in.Location = *patch.Location
	}
	if patch.StartsAt != nil {
		in.StartsAt = *patch.StartsAt
	}
	if patch.AllDay != nil {
		in.AllDay = *patch.AllDay
	}
	if patch.EndsAt != nil {
		in.EndsAt = *patch.EndsAt
	}
	clean, err := validateEventInput(in)
	if err != nil {
		return domain.Event{}, err
	}
	existing.Name, existing.Description, existing.Location = clean.Name, clean.Description, clean.Location
	existing.StartsAt, existing.AllDay, existing.EndsAt = clean.StartsAt, clean.AllDay, clean.EndsAt
	if err := s.events.Update(ctx, &existing); err != nil {
		return domain.Event{}, err
	}
	return existing, nil
}

func (s *EventService) Delete(ctx context.Context, id string) error {
	return s.events.Delete(ctx, id)
}
