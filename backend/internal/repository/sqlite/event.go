package sqlite

import (
	"context"
	"database/sql"
	"errors"

	"shale/internal/domain"
)

type EventRepo struct {
	db *DB
}

func NewEventRepo(db *DB) *EventRepo { return &EventRepo{db: db} }

const eventColumns = `id, schedule_id, name, description, location, starts_at, all_day, ends_at, created_at, updated_at`

func scanEvent(sc interface{ Scan(...any) error }) (domain.Event, error) {
	var (
		e         domain.Event
		allDay    int
		createdAt string
		updatedAt string
	)
	if err := sc.Scan(&e.ID, &e.ScheduleID, &e.Name, &e.Description, &e.Location, &e.StartsAt, &allDay, &e.EndsAt, &createdAt, &updatedAt); err != nil {
		return e, err
	}
	e.AllDay = allDay != 0
	e.CreatedAt, _ = parseTimestamp(createdAt)
	e.UpdatedAt, _ = parseTimestamp(updatedAt)
	return e, nil
}

func (r *EventRepo) ListBySchedule(ctx context.Context, scheduleID string) ([]domain.Event, error) {
	rows, err := r.db.runner(ctx).QueryContext(ctx, `
		SELECT `+eventColumns+` FROM events WHERE schedule_id = ? ORDER BY starts_at, created_at, rowid`, scheduleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectEvents(rows)
}

func (r *EventRepo) ListAll(ctx context.Context) ([]domain.Event, error) {
	rows, err := r.db.runner(ctx).QueryContext(ctx,
		`SELECT `+eventColumns+` FROM events ORDER BY schedule_id, starts_at, rowid`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectEvents(rows)
}

func collectEvents(rows *sql.Rows) ([]domain.Event, error) {
	out := []domain.Event{}
	for rows.Next() {
		e, err := scanEvent(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (r *EventRepo) GetByID(ctx context.Context, id string) (domain.Event, error) {
	row := r.db.runner(ctx).QueryRowContext(ctx,
		`SELECT `+eventColumns+` FROM events WHERE id = ?`, id)
	e, err := scanEvent(row)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Event{}, domain.ErrNotFound("event")
	}
	return e, err
}

func (r *EventRepo) Create(ctx context.Context, e *domain.Event) error {
	now := timeNow()
	e.CreatedAt, e.UpdatedAt = now, now
	if e.ID == "" {
		e.ID = domain.NewID()
	}
	_, err := r.db.runner(ctx).ExecContext(ctx, `
		INSERT INTO events (id, schedule_id, name, description, location, starts_at, all_day, ends_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		e.ID, e.ScheduleID, e.Name, e.Description, e.Location, e.StartsAt, boolToInt(e.AllDay), e.EndsAt, formatTime(now), formatTime(now))
	return mapConstraint(err, "schedule does not exist")
}

func (r *EventRepo) Update(ctx context.Context, e *domain.Event) error {
	now := timeNow()
	e.UpdatedAt = now
	res, err := r.db.runner(ctx).ExecContext(ctx, `
		UPDATE events SET name = ?, description = ?, location = ?, starts_at = ?, all_day = ?, ends_at = ?, updated_at = ?
		WHERE id = ?`,
		e.Name, e.Description, e.Location, e.StartsAt, boolToInt(e.AllDay), e.EndsAt, formatTime(now), e.ID)
	if err != nil {
		return err
	}
	if n, err := rowsAffected(res); err != nil {
		return err
	} else if n == 0 {
		return domain.ErrNotFound("event")
	}
	return nil
}

func (r *EventRepo) Delete(ctx context.Context, id string) error {
	res, err := r.db.runner(ctx).ExecContext(ctx, `DELETE FROM events WHERE id = ?`, id)
	if err != nil {
		return err
	}
	if n, err := rowsAffected(res); err != nil {
		return err
	} else if n == 0 {
		return domain.ErrNotFound("event")
	}
	return nil
}
