package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"shale/internal/domain"
)

type ScheduleRepo struct {
	db *DB
}

func NewScheduleRepo(db *DB) *ScheduleRepo { return &ScheduleRepo{db: db} }

const scheduleColumns = `id, title, description, created_at, updated_at`

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

const lastEndsAtExpr = `MAX(CASE
	WHEN e.ends_at <> '' THEN e.ends_at
	WHEN e.all_day = 1 THEN substr(e.starts_at, 1, 10) || 'T23:59'
	ELSE e.starts_at
END)`

func (r *ScheduleRepo) List(ctx context.Context) ([]domain.ScheduleWithCount, error) {
	rows, err := r.db.runner(ctx).QueryContext(ctx, `
		SELECT `+scheduleSelect+`
		FROM schedules s
		LEFT JOIN events e ON e.schedule_id = s.id
		GROUP BY s.id
		ORDER BY s.created_at DESC, s.rowid DESC`)
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

	var (
		whereSQL  string
		whereArgs []any
	)
	if search := strings.TrimSpace(q.Search); search != "" {
		like := "%" + escapeLike(search) + "%"
		whereSQL = `WHERE (
			s.title LIKE ? ESCAPE '\' COLLATE NOCASE
			OR s.description LIKE ? ESCAPE '\' COLLATE NOCASE
			OR EXISTS (
				SELECT 1 FROM events se
				WHERE se.schedule_id = s.id
				  AND (se.name LIKE ? ESCAPE '\' COLLATE NOCASE
				       OR se.location LIKE ? ESCAPE '\' COLLATE NOCASE)
			)
		)`
		whereArgs = []any{like, like, like, like}
	}

	var (
		havingSQL  string
		havingArgs []any
	)
	if !q.IncludePast {
		havingSQL = `HAVING COALESCE(` + lastEndsAtExpr + `, '') = '' OR COALESCE(` + lastEndsAtExpr + `, '') >= ?`
		havingArgs = []any{q.Now}
	}

	orderBy := scheduleOrderBy(q.Sort)
	pastFirst := `CASE WHEN COALESCE(` + lastEndsAtExpr + `, '') <> '' AND COALESCE(` + lastEndsAtExpr + `, '') < ? THEN 1 ELSE 0 END ASC, `

	limitArgs := append(append(append(append([]any{}, whereArgs...), havingArgs...), q.Now), size, (page-1)*size)

	rows, err := r.db.runner(ctx).QueryContext(ctx, `
		SELECT `+scheduleSelect+`
		FROM schedules s
		LEFT JOIN events e ON e.schedule_id = s.id
		`+whereSQL+`
		GROUP BY s.id
		`+havingSQL+`
		ORDER BY `+pastFirst+orderBy+`
		LIMIT ? OFFSET ?`, limitArgs...)
	if err != nil {
		return domain.SchedulePage{}, err
	}
	defer rows.Close()
	items, err := collectSchedules(rows)
	if err != nil {
		return domain.SchedulePage{}, err
	}

	countArgs := append(append([]any{}, whereArgs...), havingArgs...)
	var total int
	err = r.db.runner(ctx).QueryRowContext(ctx, `
		SELECT COUNT(*) FROM (
			SELECT s.id
			FROM schedules s
			LEFT JOIN events e ON e.schedule_id = s.id
			`+whereSQL+`
			GROUP BY s.id
			`+havingSQL+`
		)`, countArgs...).Scan(&total)
	if err != nil {
		return domain.SchedulePage{}, err
	}

	return domain.SchedulePage{Items: items, Total: total}, nil
}

func scheduleOrderBy(sort string) string {
	switch sort {
	case "oldest":
		return "s.created_at ASC, s.rowid ASC"
	case "title":
		return "s.title COLLATE NOCASE ASC, s.created_at DESC"
	case "title_desc":
		return "s.title COLLATE NOCASE DESC, s.created_at DESC"
	case "updated":
		return "s.updated_at DESC, s.rowid DESC"
	case "soonest":
		return "COALESCE(MIN(e.starts_at), '') = '' ASC, MIN(e.starts_at) ASC, s.created_at DESC"
	case "events":
		return "COUNT(e.id) DESC, s.created_at DESC"
	default:
		return "s.created_at DESC, s.rowid DESC"
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
		`SELECT `+`id, title, description, created_at, updated_at`+` FROM schedules WHERE id = ?`, id)
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
		VALUES (?, ?, ?, ?, ?)`,
		s.ID, s.Title, s.Description, formatTime(now), formatTime(now))
	return mapConstraint(err, "schedule conflict")
}

func (r *ScheduleRepo) Update(ctx context.Context, s *domain.Schedule) error {
	now := time.Now().UTC()
	s.UpdatedAt = now
	res, err := r.db.runner(ctx).ExecContext(ctx, `
		UPDATE schedules SET title = ?, description = ?, updated_at = ? WHERE id = ?`,
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
	res, err := r.db.runner(ctx).ExecContext(ctx, `DELETE FROM schedules WHERE id = ?`, id)
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
