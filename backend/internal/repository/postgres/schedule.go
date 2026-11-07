package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"shale/internal/domain"
)

type ScheduleRepo struct {
	db *DB
}

func NewScheduleRepo(db *DB) *ScheduleRepo { return &ScheduleRepo{db: db} }

func scanSchedule(sc interface{ Scan(...any) error }) (domain.Schedule, error) {
	var (
		s         domain.Schedule
		createdAt string
		updatedAt string
	)
	if err := sc.Scan(&s.ID, &s.Title, &s.Description, &createdAt, &updatedAt); err != nil {
		return s, err
	}
	s.CreatedAt, _ = parseTimestamp(createdAt)
	s.UpdatedAt, _ = parseTimestamp(updatedAt)
	return s, nil
}

func parseTimestamp(s string) (time.Time, bool) {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

const scheduleSelect = `s.id, s.title, s.description, s.created_at, s.updated_at,
	COUNT(e.id),
	COALESCE(MIN(e.starts_at), ''),
	COALESCE(MAX(e.starts_at), ''),
	COALESCE(` + lastEndsAtExpr + `, '')`

const eventEffectiveEndExpr = `CASE
	WHEN e.ends_at <> '' THEN substr(e.ends_at, 1, 10) || 'T23:59'
	WHEN e.starts_at IS NOT NULL AND e.starts_at <> '' THEN substr(e.starts_at, 1, 10) || 'T23:59'
	ELSE NULL
END`

const lastEndsAtExpr = `MAX(` + eventEffectiveEndExpr + `)`

func (r *ScheduleRepo) List(ctx context.Context) ([]domain.ScheduleWithCount, error) {
	rows, err := r.db.runner(ctx).QueryContext(ctx, `
		SELECT `+scheduleSelect+`
		FROM schedules s
		LEFT JOIN events e ON e.schedule_id = s.id
		GROUP BY s.id
		ORDER BY s.created_at DESC, s.id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectSchedules(rows)
}

func (r *ScheduleRepo) ListPage(ctx context.Context, q domain.ScheduleListQuery) (domain.SchedulePage, error) {
	page := q.Page
	if page < 1 {
		page = 1
	}
	size := q.PageSize
	if size < 1 {
		size = 10
	}
	if size > 100 {
		size = 100
	}

	args := []any{}
	nextArg := func(v any) string {
		args = append(args, v)
		return fmt.Sprintf("$%d", len(args))
	}

	var whereSQL string
	if search := strings.TrimSpace(q.Search); search != "" {
		like := "%" + escapeLike(search) + "%"
		p1 := nextArg(like)
		p2 := nextArg(like)
		p3 := nextArg(like)
		p4 := nextArg(like)
		whereSQL = `WHERE (
			s.title ILIKE ` + p1 + ` ESCAPE '\'
			OR s.description ILIKE ` + p2 + ` ESCAPE '\'
			OR EXISTS (
				SELECT 1 FROM events se
				WHERE se.schedule_id = s.id
				  AND (se.name ILIKE ` + p3 + ` ESCAPE '\'
				       OR se.location ILIKE ` + p4 + ` ESCAPE '\')
			)
		)`
	}

	var havingSQL string
	if !q.IncludePast {
		pNow := nextArg(q.Now)
		havingSQL = `HAVING COALESCE(` + lastEndsAtExpr + `, '') = '' OR COALESCE(` + lastEndsAtExpr + `, '') >= ` + pNow
	}

	countArgs := append([]any{}, args...)

	pPastNow := nextArg(q.Now)
	pLimit := nextArg(size)
	pOffset := nextArg((page - 1) * size)

	orderBy := scheduleOrderBy(q.Sort)
	pastFirst := `CASE WHEN COALESCE(` + lastEndsAtExpr + `, '') <> '' AND COALESCE(` + lastEndsAtExpr + `, '') < ` + pPastNow + ` THEN 1 ELSE 0 END ASC, `

	rows, err := r.db.runner(ctx).QueryContext(ctx, `
		SELECT `+scheduleSelect+`
		FROM schedules s
		LEFT JOIN events e ON e.schedule_id = s.id
		`+whereSQL+`
		GROUP BY s.id
		`+havingSQL+`
		ORDER BY `+pastFirst+orderBy+`
		LIMIT `+pLimit+` OFFSET `+pOffset, args...)
	if err != nil {
		return domain.SchedulePage{}, err
	}
	defer rows.Close()
	items, err := collectSchedules(rows)
	if err != nil {
		return domain.SchedulePage{}, err
	}

	var total int
	err = r.db.runner(ctx).QueryRowContext(ctx, `
		SELECT COUNT(*) FROM (
			SELECT s.id
			FROM schedules s
			LEFT JOIN events e ON e.schedule_id = s.id
			`+whereSQL+`
			GROUP BY s.id
			`+havingSQL+`
		) AS sub`, countArgs...).Scan(&total)
	if err != nil {
		return domain.SchedulePage{}, err
	}

	return domain.SchedulePage{Items: items, Total: total}, nil
}

func scheduleOrderBy(sort string) string {
	switch sort {
	case "oldest":
		return "s.created_at ASC, s.id ASC"
	case "title":
		return "LOWER(s.title) ASC, s.created_at DESC"
	case "title_desc":
		return "LOWER(s.title) DESC, s.created_at DESC"
	case "updated":
		return "s.updated_at DESC, s.id DESC"
	case "soonest":
		return "COALESCE(MIN(e.starts_at), '') = '' ASC, MIN(e.starts_at) ASC, s.created_at DESC"
	case "events":
		return "COUNT(e.id) DESC, s.created_at DESC"
	default:
		return "s.created_at DESC, s.id DESC"
	}
}

func escapeLike(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `%`, `\%`)
	s = strings.ReplaceAll(s, `_`, `\_`)
	return s
}

func collectSchedules(rows *sql.Rows) ([]domain.ScheduleWithCount, error) {
	out := []domain.ScheduleWithCount{}
	for rows.Next() {
		var (
			sc        domain.ScheduleWithCount
			createdAt string
			updatedAt string
		)
		if err := rows.Scan(&sc.ID, &sc.Title, &sc.Description, &createdAt, &updatedAt, &sc.EventCount, &sc.FirstStartsAt, &sc.LastStartsAt, &sc.LastEndsAt); err != nil {
			return nil, err
		}
		sc.CreatedAt, _ = parseTimestamp(createdAt)
		sc.UpdatedAt, _ = parseTimestamp(updatedAt)
		out = append(out, sc)
	}
	return out, rows.Err()
}

func (r *ScheduleRepo) GetByID(ctx context.Context, id string) (domain.Schedule, error) {
	row := r.db.runner(ctx).QueryRowContext(ctx,
		`SELECT id, title, description, created_at, updated_at FROM schedules WHERE id = $1`, id)
	s, err := scanSchedule(row)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Schedule{}, domain.ErrNotFound("schedule")
	}
	return s, err
}

func (r *ScheduleRepo) Create(ctx context.Context, s *domain.Schedule) error {
	now := time.Now().UTC()
	s.CreatedAt, s.UpdatedAt = now, now
	if s.ID == "" {
		s.ID = domain.NewID()
	}
	_, err := r.db.runner(ctx).ExecContext(ctx, `
		INSERT INTO schedules (id, title, description, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)`,
		s.ID, s.Title, s.Description, formatTime(now), formatTime(now))
	return mapConstraint(err, "schedule conflict")
}

func (r *ScheduleRepo) Update(ctx context.Context, s *domain.Schedule) error {
	now := time.Now().UTC()
	s.UpdatedAt = now
	res, err := r.db.runner(ctx).ExecContext(ctx, `
		UPDATE schedules SET title = $1, description = $2, updated_at = $3 WHERE id = $4`,
		s.Title, s.Description, formatTime(now), s.ID)
	if err != nil {
		return err
	}
	if n, err := rowsAffected(res); err != nil {
		return err
	} else if n == 0 {
		return domain.ErrNotFound("schedule")
	}
	return nil
}

func (r *ScheduleRepo) Delete(ctx context.Context, id string) error {
	res, err := r.db.runner(ctx).ExecContext(ctx, `DELETE FROM schedules WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if n, err := rowsAffected(res); err != nil {
		return err
	} else if n == 0 {
		return domain.ErrNotFound("schedule")
	}
	return nil
}

func (r *ScheduleRepo) DeleteAll(ctx context.Context) error {
	_, err := r.db.runner(ctx).ExecContext(ctx, `DELETE FROM schedules`)
	return err
}

func (r *ScheduleRepo) DeleteExpired(ctx context.Context, cutoffEndsAt string) (int64, int64, error) {
	var deletedEvents, deletedSchedules int64
	err := r.db.Do(ctx, func(ctx context.Context) error {
		rows, err := r.db.runner(ctx).QueryContext(ctx, `
			DELETE FROM events
			WHERE (CASE
				WHEN ends_at <> '' THEN substr(ends_at, 1, 10) || 'T23:59'
				ELSE substr(starts_at, 1, 10) || 'T23:59'
			END) < $1
			RETURNING schedule_id`, cutoffEndsAt)
		if err != nil {
			return err
		}
		affected := map[string]struct{}{}
		for rows.Next() {
			var sid string
			if err := rows.Scan(&sid); err != nil {
				rows.Close()
				return err
			}
			affected[sid] = struct{}{}
			deletedEvents++
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return err
		}
		rows.Close()

		for sid := range affected {
			res, err := r.db.runner(ctx).ExecContext(ctx, `
				DELETE FROM schedules
				WHERE id = $1
				  AND NOT EXISTS (SELECT 1 FROM events WHERE schedule_id = $1)`, sid)
			if err != nil {
				return err
			}
			n, err := rowsAffected(res)
			if err != nil {
				return err
			}
			deletedSchedules += n
		}
		return nil
	})
	if err != nil {
		return 0, 0, err
	}
	return deletedEvents, deletedSchedules, nil
}

func (r *ScheduleRepo) Duplicate(ctx context.Context, id string) (string, error) {
	var newID string
	err := r.db.Do(ctx, func(ctx context.Context) error {
		source, err := r.GetByID(ctx, id)
		if err != nil {
			return err
		}
		created := domain.Schedule{
			Title:       source.Title + " (copy)",
			Description: source.Description,
		}
		if err := r.Create(ctx, &created); err != nil {
			return err
		}
		events, err := r.listEventsForDuplicate(ctx, source.ID)
		if err != nil {
			return err
		}
		for _, ev := range events {
			copy := domain.Event{
				ScheduleID:  created.ID,
				Name:        ev.Name,
				Description: ev.Description,
				Location:    ev.Location,
				StartsAt:    ev.StartsAt,
				AllDay:      ev.AllDay,
				EndsAt:      ev.EndsAt,
			}
			if err := r.createEventForDuplicate(ctx, &copy); err != nil {
				return err
			}
		}
		newID = created.ID
		return nil
	})
	if err != nil {
		return "", err
	}
	return newID, nil
}
