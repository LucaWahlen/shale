package service

import (
	"context"
	"strings"

	"shale/internal/domain"
)

type ScheduleService struct {
	schedules domain.ScheduleRepository
	events    domain.EventRepository
	attendees domain.AttendeeRepository
	audit     *AuditService
	clock     Clock
}

func NewScheduleService(schedules domain.ScheduleRepository, events domain.EventRepository, attendees domain.AttendeeRepository, audit *AuditService, clock Clock) *ScheduleService {
	return &ScheduleService{schedules: schedules, events: events, attendees: attendees, audit: audit, clock: clock}
}

type ScheduleInput struct {
	Title       string
	Description string
}

type EventDetail struct {
	Event     domain.Event
	Attendees []domain.Attendee
}

type ScheduleDetail struct {
	Schedule domain.Schedule
	Events   []EventDetail
	IsPast   bool
}

type SchedulePageDetail struct {
	Items []ScheduleDetail
	Total int
}

type PublicAttendee struct {
	ID   string
	Name string
}

type PublicEvent struct {
	Event      domain.Event
	Attendees  []PublicAttendee
	Attendable bool
}

type PublicSchedule struct {
	Schedule domain.Schedule
	Events   []PublicEvent
}

const (
	maxTitleRunes       = 200
	maxDescriptionRunes = 5000
)

func validateTitle(title string) (string, error) {
	trimmed := strings.TrimSpace(title)
	if trimmed == "" {
		return "", domain.NewError(domain.KindInvalid, "title must not be empty")
	}
	if runeLen(trimmed) > maxTitleRunes {
		return "", domain.Errorf(domain.KindInvalid, "title must be at most %d characters", maxTitleRunes)
	}
	return trimmed, nil
}

func validateDescription(desc string) (string, error) {
	trimmed := strings.TrimSpace(desc)
	if runeLen(trimmed) > maxDescriptionRunes {
		return "", domain.Errorf(domain.KindInvalid, "description must be at most %d characters", maxDescriptionRunes)
	}
	return trimmed, nil
}

func (s *ScheduleService) List(ctx context.Context) ([]domain.ScheduleWithCount, error) {
	return s.schedules.List(ctx)
}

func (s *ScheduleService) ListDetailed(ctx context.Context, q domain.ScheduleListQuery) (SchedulePageDetail, error) {
	q.Now = s.clock().Format(domain.StartsAtFormat)
	page, err := s.schedules.ListPage(ctx, q)
	if err != nil {
		return SchedulePageDetail{}, err
	}
	out := SchedulePageDetail{
		Items: make([]ScheduleDetail, 0, len(page.Items)),
		Total: page.Total,
	}
	for _, sc := range page.Items {
		sched := domain.Schedule{
			ID:          sc.ID,
			Title:       sc.Title,
			Description: sc.Description,
			CreatedAt:   sc.CreatedAt,
			UpdatedAt:   sc.UpdatedAt,
		}
		detail, err := s.assemble(ctx, sched)
		if err != nil {
			return SchedulePageDetail{}, err
		}
		detail.IsPast = scheduleIsPast(detail.Events, q.Now)
		out.Items = append(out.Items, detail)
	}
	return out, nil
}

func scheduleIsPast(events []EventDetail, now string) bool {
	if len(events) == 0 {
		return false
	}
	lastEnd := ""
	for _, ed := range events {
		end := eventEndAt(ed.Event)
		if end > lastEnd {
			lastEnd = end
		}
	}
	return lastEnd != "" && lastEnd < now
}

func eventEndAt(e domain.Event) string {
	if e.EndsAt != "" {
		return e.EndsAt
	}
	if e.AllDay && len(e.StartsAt) >= 10 {
		return e.StartsAt[:10] + "T23:59"
	}
	return e.StartsAt
}

func (s *ScheduleService) Get(ctx context.Context, id string) (ScheduleDetail, error) {
	sched, err := s.schedules.GetByID(ctx, id)
	if err != nil {
		return ScheduleDetail{}, err
	}
	return s.assemble(ctx, sched)
}

func (s *ScheduleService) GetPublic(ctx context.Context, id string) (PublicSchedule, error) {
	sched, err := s.schedules.GetByID(ctx, id)
	if err != nil {
		return PublicSchedule{}, err
	}
	detail, err := s.assemble(ctx, sched)
	if err != nil {
		return PublicSchedule{}, err
	}
	now := s.clock()
	out := PublicSchedule{Schedule: sched, Events: make([]PublicEvent, 0, len(detail.Events))}
	for _, ed := range detail.Events {
		pe := PublicEvent{
			Event:      ed.Event,
			Attendable: domain.Attendable(ed.Event.StartsAt, now),
			Attendees:  make([]PublicAttendee, 0, len(ed.Attendees)),
		}
		for _, a := range ed.Attendees {
			pe.Attendees = append(pe.Attendees, PublicAttendee{ID: a.ID, Name: a.Name})
		}
		out.Events = append(out.Events, pe)
	}
	return out, nil
}

func (s *ScheduleService) assemble(ctx context.Context, sched domain.Schedule) (ScheduleDetail, error) {
	events, err := s.events.ListBySchedule(ctx, sched.ID)
	if err != nil {
		return ScheduleDetail{}, err
	}
	out := ScheduleDetail{Schedule: sched, Events: make([]EventDetail, 0, len(events))}
	for _, ev := range events {
		atts, err := s.attendees.ListByEvent(ctx, ev.ID)
		if err != nil {
			return ScheduleDetail{}, err
		}
		out.Events = append(out.Events, EventDetail{Event: ev, Attendees: atts})
	}
	return out, nil
}

func (s *ScheduleService) Create(ctx context.Context, in ScheduleInput) (domain.Schedule, error) {
	title, err := validateTitle(in.Title)
	if err != nil {
		return domain.Schedule{}, err
	}
	desc, err := validateDescription(in.Description)
	if err != nil {
		return domain.Schedule{}, err
	}
	sched := domain.Schedule{Title: title, Description: desc}
	if err := s.schedules.Create(ctx, &sched); err != nil {
		return domain.Schedule{}, err
	}
	s.audit.Record(ctx, AuditInput{
		Action:        domain.AuditActionScheduleCreated,
		Actor:         domain.AuditActorAdmin,
		ScheduleID:    sched.ID,
		ScheduleTitle: sched.Title,
	})
	return sched, nil
}

func (s *ScheduleService) Update(ctx context.Context, id string, in ScheduleInput) (domain.Schedule, error) {
	existing, err := s.schedules.GetByID(ctx, id)
	if err != nil {
		return domain.Schedule{}, err
	}
	title, err := validateTitle(in.Title)
	if err != nil {
		return domain.Schedule{}, err
	}
	desc, err := validateDescription(in.Description)
	if err != nil {
		return domain.Schedule{}, err
	}
	existing.Title, existing.Description = title, desc
	if err := s.schedules.Update(ctx, &existing); err != nil {
		return domain.Schedule{}, err
	}
	s.audit.Record(ctx, AuditInput{
		Action:        domain.AuditActionScheduleUpdated,
		Actor:         domain.AuditActorAdmin,
		ScheduleID:    existing.ID,
		ScheduleTitle: existing.Title,
	})
	return existing, nil
}

func (s *ScheduleService) Delete(ctx context.Context, id string) error {
	existing, err := s.schedules.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if err := s.schedules.Delete(ctx, id); err != nil {
		return err
	}
	s.audit.Record(ctx, AuditInput{
		Action:        domain.AuditActionScheduleDeleted,
		Actor:         domain.AuditActorAdmin,
		ScheduleID:    existing.ID,
		ScheduleTitle: existing.Title,
	})
	return nil
}

func (s *ScheduleService) Duplicate(ctx context.Context, id string, newTitle string) (domain.Schedule, error) {
	source, err := s.schedules.GetByID(ctx, id)
	if err != nil {
		return domain.Schedule{}, err
	}
	title, err := validateTitle(newTitleForDuplicate(source.Title, newTitle))
	if err != nil {
		return domain.Schedule{}, err
	}
	newID, err := s.schedules.Duplicate(ctx, source.ID)
	if err != nil {
		return domain.Schedule{}, err
	}
	created, err := s.schedules.GetByID(ctx, newID)
	if err != nil {
		return domain.Schedule{}, err
	}
	if title != source.Title {
		if err := s.schedules.Update(ctx, &created); err != nil {
			return domain.Schedule{}, err
		}
	}
	s.audit.Record(ctx, AuditInput{
		Action:        domain.AuditActionScheduleDuplicate,
		Actor:         domain.AuditActorAdmin,
		ScheduleID:    created.ID,
		ScheduleTitle: created.Title,
		Detail:        source.Title,
	})
	return created, nil
}

func newTitleForDuplicate(sourceTitle, requested string) string {
	if strings.TrimSpace(requested) != "" {
		return requested
	}
	return sourceTitle + " (copy)"
}

func runeLen(s string) int {
	n := 0
	for range s {
		n++
	}
	return n
}
