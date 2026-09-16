package sqlite

import (
	"context"
	"database/sql"
	"errors"

	"shale/internal/domain"
)

type AttendeeRepo struct {
	db *DB
}

func NewAttendeeRepo(db *DB) *AttendeeRepo { return &AttendeeRepo{db: db} }

const attendeeColumns = `id, event_id, name, normalized_name, manage_token, created_at`

func scanAttendee(sc interface{ Scan(...any) error }) (domain.Attendee, error) {
	var (
		a         domain.Attendee
		createdAt string
	)
	if err := sc.Scan(&a.ID, &a.EventID, &a.Name, &a.NormalizedName, &a.ManageToken, &createdAt); err != nil {
		return a, err
	}
	a.CreatedAt, _ = parseTimestamp(createdAt)
	return a, nil
}

func (r *AttendeeRepo) ListByEvent(ctx context.Context, eventID string) ([]domain.Attendee, error) {
	rows, err := r.db.runner(ctx).QueryContext(ctx, `
		SELECT `+attendeeColumns+` FROM attendees WHERE event_id = ? ORDER BY created_at, rowid`, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.Attendee{}
	for rows.Next() {
		a, err := scanAttendee(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *AttendeeRepo) GetByID(ctx context.Context, id string) (domain.Attendee, error) {
	row := r.db.runner(ctx).QueryRowContext(ctx,
		`SELECT `+attendeeColumns+` FROM attendees WHERE id = ?`, id)
	a, err := scanAttendee(row)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Attendee{}, domain.ErrNotFound("attendee")
	}
	return a, err
}

func (r *AttendeeRepo) FindByEventAndNormalizedName(ctx context.Context, eventID string, normalized string) (domain.Attendee, error) {
	row := r.db.runner(ctx).QueryRowContext(ctx,
		`SELECT `+attendeeColumns+` FROM attendees WHERE event_id = ? AND normalized_name = ?`, eventID, normalized)
	a, err := scanAttendee(row)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Attendee{}, domain.ErrNotFound("attendee")
	}
	return a, err
}

func (r *AttendeeRepo) Create(ctx context.Context, a *domain.Attendee) error {
	now := timeNow()
	a.CreatedAt = now
	if a.ID == "" {
		a.ID = domain.NewID()
	}
	_, err := r.db.runner(ctx).ExecContext(ctx, `
		INSERT INTO attendees (id, event_id, name, normalized_name, manage_token, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		a.ID, a.EventID, a.Name, a.NormalizedName, a.ManageToken, formatTime(now))
	return mapConstraint(err, "this name is already taken for this event")
}

func (r *AttendeeRepo) UpdateToken(ctx context.Context, id string, token string) error {
	res, err := r.db.runner(ctx).ExecContext(ctx,
		`UPDATE attendees SET manage_token = ? WHERE id = ?`, token, id)
	if err := mapConstraint(err, "token conflict"); err != nil {
		return err
	}
	if n, err := rowsAffected(res); err != nil {
		return err
	} else if n == 0 {
		return domain.ErrNotFound("attendee")
	}
	return nil
}

func (r *AttendeeRepo) Delete(ctx context.Context, id string) error {
	res, err := r.db.runner(ctx).ExecContext(ctx, `DELETE FROM attendees WHERE id = ?`, id)
	if err != nil {
		return err
	}
	if n, err := rowsAffected(res); err != nil {
		return err
	} else if n == 0 {
		return domain.ErrNotFound("attendee")
	}
	return nil
}
