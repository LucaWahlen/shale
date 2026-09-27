package domain

import "time"

type AuditEntry struct {
	ID            string
	CreatedAt     time.Time
	Action        string
	Actor         string
	ActorName     string
	ScheduleID    string
	ScheduleTitle string
	EventID       string
	EventName     string
	Detail        string
}

type AuditListQuery struct {
	Action     string
	ScheduleID string
	Search     string
	Page       int
	PageSize   int
}

type AuditPage struct {
	Items []AuditEntry
	Total int
}

const (
	AuditActorAdmin  = "admin"
	AuditActorPublic = "public"
)

const (
	AuditActionScheduleCreated   = "schedule.created"
	AuditActionScheduleUpdated   = "schedule.updated"
	AuditActionScheduleDeleted   = "schedule.deleted"
	AuditActionScheduleDuplicate = "schedule.duplicated"
	AuditActionEventCreated      = "event.created"
	AuditActionEventUpdated      = "event.updated"
	AuditActionEventDeleted      = "event.deleted"
	AuditActionAttendeeAdded     = "attendee.added"
	AuditActionAttendeeRemoved   = "attendee.removed"
)

func ValidAuditAction(a string) bool {
	switch a {
	case AuditActionScheduleCreated, AuditActionScheduleUpdated, AuditActionScheduleDeleted,
		AuditActionScheduleDuplicate, AuditActionEventCreated, AuditActionEventUpdated,
		AuditActionEventDeleted, AuditActionAttendeeAdded, AuditActionAttendeeRemoved:
		return true
	}
	return false
}
