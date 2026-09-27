package service

import (
	"context"
	"crypto/subtle"

	"shale/internal/domain"
)

type AttendeeService struct {
	schedules domain.ScheduleRepository
	events    domain.EventRepository
	attendees domain.AttendeeRepository
	audit     *AuditService
	clock     Clock
}

func NewAttendeeService(schedules domain.ScheduleRepository, events domain.EventRepository, attendees domain.AttendeeRepository, audit *AuditService, clock Clock) *AttendeeService {
	return &AttendeeService{schedules: schedules, events: events, attendees: attendees, audit: audit, clock: clock}
}

type AttendResult struct {
	Attendee domain.Attendee
	Token    string
	Created  bool
}

func (s *AttendeeService) Attend(ctx context.Context, scheduleID string, eventID string, rawName string) (AttendResult, error) {
	sched, err := s.schedules.GetByID(ctx, scheduleID)
	if err != nil {
		return AttendResult{}, err
	}
	ev, err := s.lookupEvent(ctx, sched.ID, eventID)
	if err != nil {
		return AttendResult{}, err
	}
	if !domain.Attendable(ev.StartsAt, s.clock()) {
		return AttendResult{}, domain.NewError(domain.KindNotAttendable, "attendance is closed for this event")
	}
	name, err := domain.ValidateName(rawName)
	if err != nil {
		return AttendResult{}, err
	}
	normalized := domain.NormalizeName(name)
	token, err := domain.NewManageToken()
	if err != nil {
		return AttendResult{}, err
	}
	existing, err := s.attendees.FindByEventAndNormalizedName(ctx, ev.ID, normalized)
	if err == nil {
		if err := s.attendees.UpdateToken(ctx, existing.ID, token); err != nil {
			return AttendResult{}, err
		}
		existing.ManageToken = token
		return AttendResult{Attendee: existing, Token: token, Created: false}, nil
	}
	if k, ok := domain.KindOf(err); ok && k != domain.KindNotFound {
		return AttendResult{}, err
	}
	att := domain.Attendee{
		EventID:        ev.ID,
		Name:           name,
		NormalizedName: normalized,
		ManageToken:    token,
	}
	if err := s.attendees.Create(ctx, &att); err != nil {

		if k, ok := domain.KindOf(err); ok && k == domain.KindConflict {
			existing, ferr := s.attendees.FindByEventAndNormalizedName(ctx, ev.ID, normalized)
			if ferr != nil {
				return AttendResult{}, ferr
			}
			if err := s.attendees.UpdateToken(ctx, existing.ID, token); err != nil {
				return AttendResult{}, err
			}
			existing.ManageToken = token
			return AttendResult{Attendee: existing, Token: token, Created: false}, nil
		}
		return AttendResult{}, err
	}
	s.audit.Record(ctx, AuditInput{
		Action:        domain.AuditActionAttendeeAdded,
		Actor:         domain.AuditActorPublic,
		ActorName:     att.Name,
		ScheduleID:    sched.ID,
		ScheduleTitle: sched.Title,
		EventID:       ev.ID,
		EventName:     ev.Name,
	})
	return AttendResult{Attendee: att, Token: token, Created: true}, nil
}

func (s *AttendeeService) Revert(ctx context.Context, scheduleID string, eventID, attendeeID string, token string) error {
	sched, err := s.schedules.GetByID(ctx, scheduleID)
	if err != nil {
		return err
	}
	ev, err := s.lookupEvent(ctx, sched.ID, eventID)
	if err != nil {
		return err
	}
	att, err := s.attendees.GetByID(ctx, attendeeID)
	if err != nil {
		return err
	}
	if att.EventID != ev.ID {
		return domain.ErrNotFound("attendee")
	}
	if !domain.Attendable(ev.StartsAt, s.clock()) {
		return domain.NewError(domain.KindNotAttendable, "attendance is closed for this event")
	}
	if subtle.ConstantTimeCompare([]byte(att.ManageToken), []byte(token)) != 1 {
		return domain.NewError(domain.KindForbidden, "manage token does not match")
	}
	if err := s.attendees.Delete(ctx, att.ID); err != nil {
		return err
	}
	s.audit.Record(ctx, AuditInput{
		Action:        domain.AuditActionAttendeeRemoved,
		Actor:         domain.AuditActorPublic,
		ActorName:     att.Name,
		ScheduleID:    sched.ID,
		ScheduleTitle: sched.Title,
		EventID:       ev.ID,
		EventName:     ev.Name,
	})
	return nil
}

func (s *AttendeeService) AdminAdd(ctx context.Context, eventID string, rawName string) (domain.Attendee, error) {
	ev, err := s.events.GetByID(ctx, eventID)
	if err != nil {
		return domain.Attendee{}, err
	}
	sched, err := s.schedules.GetByID(ctx, ev.ScheduleID)
	if err != nil {
		return domain.Attendee{}, err
	}
	name, err := domain.ValidateName(rawName)
	if err != nil {
		return domain.Attendee{}, err
	}
	normalized := domain.NormalizeName(name)
	token, err := domain.NewManageToken()
	if err != nil {
		return domain.Attendee{}, err
	}
	existing, err := s.attendees.FindByEventAndNormalizedName(ctx, ev.ID, normalized)
	if err == nil {
		return existing, nil
	}
	if k, ok := domain.KindOf(err); ok && k != domain.KindNotFound {
		return domain.Attendee{}, err
	}
	att := domain.Attendee{
		EventID:        ev.ID,
		Name:           name,
		NormalizedName: normalized,
		ManageToken:    token,
	}
	if err := s.attendees.Create(ctx, &att); err != nil {
		if k, ok := domain.KindOf(err); ok && k == domain.KindConflict {
			return s.attendees.FindByEventAndNormalizedName(ctx, ev.ID, normalized)
		}
		return domain.Attendee{}, err
	}
	s.audit.Record(ctx, AuditInput{
		Action:        domain.AuditActionAttendeeAdded,
		Actor:         domain.AuditActorAdmin,
		ActorName:     att.Name,
		ScheduleID:    sched.ID,
		ScheduleTitle: sched.Title,
		EventID:       ev.ID,
		EventName:     ev.Name,
	})
	return att, nil
}

func (s *AttendeeService) AdminRemove(ctx context.Context, attendeeID string) error {
	att, err := s.attendees.GetByID(ctx, attendeeID)
	if err != nil {
		return err
	}
	ev, err := s.events.GetByID(ctx, att.EventID)
	if err != nil {
		return err
	}
	sched, err := s.schedules.GetByID(ctx, ev.ScheduleID)
	if err != nil {
		return err
	}
	if err := s.attendees.Delete(ctx, attendeeID); err != nil {
		return err
	}
	s.audit.Record(ctx, AuditInput{
		Action:        domain.AuditActionAttendeeRemoved,
		Actor:         domain.AuditActorAdmin,
		ActorName:     att.Name,
		ScheduleID:    sched.ID,
		ScheduleTitle: sched.Title,
		EventID:       ev.ID,
		EventName:     ev.Name,
	})
	return nil
}

func (s *AttendeeService) lookupEvent(ctx context.Context, scheduleID, eventID string) (domain.Event, error) {
	ev, err := s.events.GetByID(ctx, eventID)
	if err != nil {
		return domain.Event{}, err
	}
	if ev.ScheduleID != scheduleID {
		return domain.Event{}, domain.ErrNotFound("event")
	}
	return ev, nil
}
