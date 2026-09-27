package sqlite

import (
	"context"
	"strings"

	"shale/internal/domain"
)

type AuditRepo struct {
	db *DB
}

func NewAuditRepo(db *DB) *AuditRepo { return &AuditRepo{db: db} }

const auditColumns = `id, created_at, action, actor, actor_name, schedule_id, schedule_title, event_id, event_name, detail`

func scanAudit(sc interface{ Scan(...any) error }) (domain.AuditEntry, error) {
	var (
		e         domain.AuditEntry
		createdAt string
	)
	if err := sc.Scan(&e.ID, &createdAt, &e.Action, &e.Actor, &e.ActorName, &e.ScheduleID, &e.ScheduleTitle, &e.EventID, &e.EventName, &e.Detail); err != nil {
		return e, err
	}
	e.CreatedAt, _ = parseTimestamp(createdAt)
	return e, nil
}

func (r *AuditRepo) Append(ctx context.Context, e *domain.AuditEntry) error {
	if e.ID == "" {
		e.ID = domain.NewID()
	}
	now := timeNow()
	e.CreatedAt = now
	_, err := r.db.runner(ctx).ExecContext(ctx, `
		INSERT INTO audit_log (id, created_at, action, actor, actor_name, schedule_id, schedule_title, event_id, event_name, detail)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		e.ID, formatTime(now), e.Action, e.Actor, e.ActorName, e.ScheduleID, e.ScheduleTitle, e.EventID, e.EventName, e.Detail)
	return err
}

func (r *AuditRepo) List(ctx context.Context, q domain.AuditListQuery) (domain.AuditPage, error) {
	page := q.Page
	if page < 1 {
		page = 1
	}
	size := q.PageSize
	if size < 1 {
		size = 25
	}
	if size > 100 {
		size = 100
	}

	where := []string{"1 = 1"}
	args := []any{}
	if q.Action != "" {
		where = append(where, "action = ?")
		args = append(args, q.Action)
	}
	if q.ScheduleID != "" {
		where = append(where, "schedule_id = ?")
		args = append(args, q.ScheduleID)
	}
	if search := strings.TrimSpace(q.Search); search != "" {
		like := "%" + escapeLike(search) + "%"
		where = append(where, `(actor_name LIKE ? ESCAPE '\' COLLATE NOCASE OR event_name LIKE ? ESCAPE '\' COLLATE NOCASE OR schedule_title LIKE ? ESCAPE '\' COLLATE NOCASE OR detail LIKE ? ESCAPE '\' COLLATE NOCASE)`)
		args = append(args, like, like, like, like)
	}
	whereSQL := "WHERE " + strings.Join(where, " AND ")

	rows, err := r.db.runner(ctx).QueryContext(ctx, `
		SELECT `+auditColumns+`
		FROM audit_log
		`+whereSQL+`
		ORDER BY created_at DESC, rowid DESC
		LIMIT ? OFFSET ?`,
		append(append([]any{}, args...), size, (page-1)*size)...)
	if err != nil {
		return domain.AuditPage{}, err
	}
	defer rows.Close()
	items := []domain.AuditEntry{}
	for rows.Next() {
		e, err := scanAudit(rows)
		if err != nil {
			return domain.AuditPage{}, err
		}
		items = append(items, e)
	}
	if err := rows.Err(); err != nil {
		return domain.AuditPage{}, err
	}

	var total int
	if err := r.db.runner(ctx).QueryRowContext(ctx,
		`SELECT COUNT(*) FROM audit_log `+whereSQL, args...).Scan(&total); err != nil {
		return domain.AuditPage{}, err
	}

	return domain.AuditPage{Items: items, Total: total}, nil
}

func (r *AuditRepo) DeleteBefore(ctx context.Context, cutoff string) (int64, error) {
	res, err := r.db.runner(ctx).ExecContext(ctx, `DELETE FROM audit_log WHERE created_at < ?`, cutoff)
	if err != nil {
		return 0, err
	}
	return rowsAffected(res)
}
