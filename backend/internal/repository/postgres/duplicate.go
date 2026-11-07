package postgres

import (
	"context"

	"shale/internal/domain"
)

func (r *ScheduleRepo) listEventsForDuplicate(ctx context.Context, scheduleID string) ([]domain.Event, error) {
	rows, err := r.db.runner(ctx).QueryContext(ctx, `
		SELECT id, schedule_id, name, description, location, starts_at, all_day, ends_at, created_at, updated_at
		FROM events WHERE schedule_id = $1 ORDER BY starts_at, id`, scheduleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.Event{}
	for rows.Next() {
		var (
			e         domain.Event
			allDay    int
			createdAt string
			updatedAt string
		)
		if err := rows.Scan(&e.ID, &e.ScheduleID, &e.Name, &e.Description, &e.Location, &e.StartsAt, &allDay, &e.EndsAt, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		e.AllDay = allDay != 0
		out = append(out, e)
	}
	return out, rows.Err()
}

func (r *ScheduleRepo) createEventForDuplicate(ctx context.Context, e *domain.Event) error {
	if e.ID == "" {
		e.ID = domain.NewID()
	}
	now := formatTime(timeNow())
	_, err := r.db.runner(ctx).ExecContext(ctx, `
		INSERT INTO events (id, schedule_id, name, description, location, starts_at, all_day, ends_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		e.ID, e.ScheduleID, e.Name, e.Description, e.Location, e.StartsAt, boolToInt(e.AllDay), e.EndsAt, now, now)
	return err
}
