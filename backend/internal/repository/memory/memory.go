package memory

import (
	"cmp"
	"context"
	"maps"
	"slices"
	"strings"
	"sync"
	"time"

	"shale/internal/domain"
)

type Store struct {
	mu        sync.Mutex
	seq       int64
	schedules map[string]scheduleRecord
	events    map[string]eventRecord
	attendees map[string]attendeeRecord
	settings  map[string]string
}

type scheduleRecord struct {
	domain.Schedule
	seq int64
}

type eventRecord struct {
	domain.Event
	seq int64
}

type attendeeRecord struct {
	domain.Attendee
	seq int64
}

func New() *Store {
	return &Store{
		schedules: make(map[string]scheduleRecord),
		events:    make(map[string]eventRecord),
		attendees: make(map[string]attendeeRecord),
		settings: map[string]string{
			domain.SettingDefaultLanguage: domain.LanguageEN,
		},
	}
}

func (s *Store) Do(ctx context.Context, fn func(ctx context.Context) error) error {
	s.mu.Lock()
	snapSchedules := maps.Clone(s.schedules)
	snapEvents := maps.Clone(s.events)
	snapAttendees := maps.Clone(s.attendees)
	snapSettings := maps.Clone(s.settings)
	snapSeq := s.seq
	s.mu.Unlock()

	if err := fn(ctx); err != nil {
		s.mu.Lock()
		s.schedules = snapSchedules
		s.events = snapEvents
		s.attendees = snapAttendees
		s.settings = snapSettings
		s.seq = snapSeq
		s.mu.Unlock()
		return err
	}
	return nil
}

type ScheduleRepo struct{ s *Store }
type EventRepo struct{ s *Store }
type AttendeeRepo struct{ s *Store }
type SettingsRepo struct{ s *Store }

func NewScheduleRepo(s *Store) *ScheduleRepo { return &ScheduleRepo{s: s} }
func NewEventRepo(s *Store) *EventRepo       { return &EventRepo{s: s} }
func NewAttendeeRepo(s *Store) *AttendeeRepo { return &AttendeeRepo{s: s} }
func NewSettingsRepo(s *Store) *SettingsRepo { return &SettingsRepo{s: s} }

func (r *ScheduleRepo) withCountsLocked() []domain.ScheduleWithCount {
	out := make([]domain.ScheduleWithCount, 0, len(r.s.schedules))
	for _, sr := range r.s.schedules {
		sc := domain.ScheduleWithCount{Schedule: sr.Schedule}
		for _, er := range r.s.events {
			if er.ScheduleID != sr.ID {
				continue
			}
			sc.EventCount++
			if sc.FirstStartsAt == "" || er.StartsAt < sc.FirstStartsAt {
				sc.FirstStartsAt = er.StartsAt
			}
			if sc.LastStartsAt == "" || er.StartsAt > sc.LastStartsAt {
				sc.LastStartsAt = er.StartsAt
			}
			end := domain.EventEndAt(er.StartsAt, er.EndsAt)
			if end > sc.LastEndsAt {
				sc.LastEndsAt = end
			}
		}
		out = append(out, sc)
	}
	return out
}

func (r *ScheduleRepo) List(ctx context.Context) ([]domain.ScheduleWithCount, error) {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()

	items := r.withCountsLocked()
	slices.SortFunc(items, func(a, b domain.ScheduleWithCount) int {
		if c := b.CreatedAt.Compare(a.CreatedAt); c != 0 {
			return c
		}
		return cmp.Compare(r.s.schedules[b.ID].seq, r.s.schedules[a.ID].seq)
	})
	return items, nil
}

func (r *ScheduleRepo) ListPage(ctx context.Context, q domain.ScheduleListQuery) (domain.SchedulePage, error) {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()

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

	search := strings.ToLower(strings.TrimSpace(q.Search))
	all := r.withCountsLocked()
	filtered := make([]domain.ScheduleWithCount, 0, len(all))
	for _, sc := range all {
		if search != "" {
			matched := strings.Contains(strings.ToLower(sc.Title), search) ||
				strings.Contains(strings.ToLower(sc.Description), search)
			if !matched {
				for _, er := range r.s.events {
					if er.ScheduleID == sc.ID &&
						(strings.Contains(strings.ToLower(er.Name), search) ||
							strings.Contains(strings.ToLower(er.Location), search)) {
						matched = true
						break
					}
				}
			}
			if !matched {
				continue
			}
		}
		if !q.IncludePast && sc.LastEndsAt != "" && sc.LastEndsAt < q.Now {
			continue
		}
		filtered = append(filtered, sc)
	}

	isPast := func(sc domain.ScheduleWithCount) int {
		if sc.LastEndsAt != "" && sc.LastEndsAt < q.Now {
			return 1
		}
		return 0
	}

	slices.SortFunc(filtered, func(a, b domain.ScheduleWithCount) int {
		if c := cmp.Compare(isPast(a), isPast(b)); c != 0 {
			return c
		}
		seqA := r.s.schedules[a.ID].seq
		seqB := r.s.schedules[b.ID].seq
		switch q.Sort {
		case "oldest":
			if c := a.CreatedAt.Compare(b.CreatedAt); c != 0 {
				return c
			}
			return cmp.Compare(seqA, seqB)
		case "title":
			if c := cmp.Compare(strings.ToLower(a.Title), strings.ToLower(b.Title)); c != 0 {
				return c
			}
			return b.CreatedAt.Compare(a.CreatedAt)
		case "title_desc":
			if c := cmp.Compare(strings.ToLower(b.Title), strings.ToLower(a.Title)); c != 0 {
				return c
			}
			return b.CreatedAt.Compare(a.CreatedAt)
		case "updated":
			if c := b.UpdatedAt.Compare(a.UpdatedAt); c != 0 {
				return c
			}
			return cmp.Compare(seqB, seqA)
		case "soonest":
			emptyA, emptyB := a.FirstStartsAt == "", b.FirstStartsAt == ""
			if emptyA != emptyB {
				if emptyA {
					return 1
				}
				return -1
			}
			if c := cmp.Compare(a.FirstStartsAt, b.FirstStartsAt); c != 0 {
				return c
			}
			return b.CreatedAt.Compare(a.CreatedAt)
		case "events":
			if c := cmp.Compare(b.EventCount, a.EventCount); c != 0 {
				return c
			}
			return b.CreatedAt.Compare(a.CreatedAt)
		default:
			if c := b.CreatedAt.Compare(a.CreatedAt); c != 0 {
				return c
			}
			return cmp.Compare(seqB, seqA)
		}
	})

	total := len(filtered)
	start := (page - 1) * size
	if start >= total {
		return domain.SchedulePage{Items: []domain.ScheduleWithCount{}, Total: total}, nil
	}
	end := start + size
	if end > total {
		end = total
	}
	return domain.SchedulePage{Items: filtered[start:end], Total: total}, nil
}

func (r *ScheduleRepo) GetByID(ctx context.Context, id string) (domain.Schedule, error) {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	sr, ok := r.s.schedules[id]
	if !ok {
		return domain.Schedule{}, domain.ErrNotFound("schedule")
	}
	return sr.Schedule, nil
}

func (r *ScheduleRepo) Create(ctx context.Context, s *domain.Schedule) error {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	now := time.Now().UTC().Truncate(time.Second)
	s.CreatedAt, s.UpdatedAt = now, now
	if s.ID == "" {
		s.ID = domain.NewID()
	}
	if _, exists := r.s.schedules[s.ID]; exists {
		return domain.NewError(domain.KindConflict, "schedule conflict")
	}
	r.s.seq++
	r.s.schedules[s.ID] = scheduleRecord{Schedule: *s, seq: r.s.seq}
	return nil
}

func (r *ScheduleRepo) Update(ctx context.Context, s *domain.Schedule) error {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	sr, ok := r.s.schedules[s.ID]
	if !ok {
		return domain.ErrNotFound("schedule")
	}
	now := time.Now().UTC().Truncate(time.Second)
	s.UpdatedAt = now
	sr.Title = s.Title
	sr.Description = s.Description
	sr.UpdatedAt = now
	r.s.schedules[s.ID] = sr
	return nil
}

func (r *ScheduleRepo) Delete(ctx context.Context, id string) error {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	if _, ok := r.s.schedules[id]; !ok {
		return domain.ErrNotFound("schedule")
	}
	delete(r.s.schedules, id)
	for eid, er := range r.s.events {
		if er.ScheduleID == id {
			delete(r.s.events, eid)
			for aid, ar := range r.s.attendees {
				if ar.EventID == eid {
					delete(r.s.attendees, aid)
				}
			}
		}
	}
	return nil
}

func (r *ScheduleRepo) DeleteAll(ctx context.Context) error {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	clear(r.s.schedules)
	clear(r.s.events)
	clear(r.s.attendees)
	return nil
}

func (r *ScheduleRepo) DeleteExpired(ctx context.Context, cutoffEndsAt string) (int64, int64, error) {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()

	var deletedEvents, deletedSchedules int64
	affected := make(map[string]struct{})
	for eid, er := range r.s.events {
		end := domain.EventEndAt(er.StartsAt, er.EndsAt)
		if end != "" && end < cutoffEndsAt {
			affected[er.ScheduleID] = struct{}{}
			delete(r.s.events, eid)
			deletedEvents++
			for aid, ar := range r.s.attendees {
				if ar.EventID == eid {
					delete(r.s.attendees, aid)
				}
			}
		}
	}

	for sid := range affected {
		hasRemaining := false
		for _, er := range r.s.events {
			if er.ScheduleID == sid {
				hasRemaining = true
				break
			}
		}
		if !hasRemaining {
			if _, ok := r.s.schedules[sid]; ok {
				delete(r.s.schedules, sid)
				deletedSchedules++
			}
		}
	}
	return deletedEvents, deletedSchedules, nil
}

func (r *ScheduleRepo) Duplicate(ctx context.Context, id string) (string, error) {
	source, err := r.GetByID(ctx, id)
	if err != nil {
		return "", err
	}
	created := domain.Schedule{
		Title:       source.Title + " (copy)",
		Description: source.Description,
	}
	if err := r.Create(ctx, &created); err != nil {
		return "", err
	}

	r.s.mu.Lock()
	evs := make([]eventRecord, 0)
	for _, er := range r.s.events {
		if er.ScheduleID == source.ID {
			evs = append(evs, er)
		}
	}
	slices.SortFunc(evs, func(a, b eventRecord) int {
		if c := cmp.Compare(a.StartsAt, b.StartsAt); c != 0 {
			return c
		}
		return cmp.Compare(a.seq, b.seq)
	})
	now := time.Now().UTC().Truncate(time.Second)
	for _, er := range evs {
		r.s.seq++
		ev := domain.Event{
			ID:          domain.NewID(),
			ScheduleID:  created.ID,
			Name:        er.Name,
			Description: er.Description,
			Location:    er.Location,
			StartsAt:    er.StartsAt,
			AllDay:      er.AllDay,
			EndsAt:      er.EndsAt,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		r.s.events[ev.ID] = eventRecord{Event: ev, seq: r.s.seq}
	}
	r.s.mu.Unlock()
	return created.ID, nil
}

func (r *EventRepo) ListBySchedule(ctx context.Context, scheduleID string) ([]domain.Event, error) {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	recs := make([]eventRecord, 0)
	for _, er := range r.s.events {
		if er.ScheduleID == scheduleID {
			recs = append(recs, er)
		}
	}
	slices.SortFunc(recs, func(a, b eventRecord) int {
		if c := cmp.Compare(a.StartsAt, b.StartsAt); c != 0 {
			return c
		}
		if c := a.CreatedAt.Compare(b.CreatedAt); c != 0 {
			return c
		}
		return cmp.Compare(a.seq, b.seq)
	})
	out := make([]domain.Event, len(recs))
	for i, rec := range recs {
		out[i] = rec.Event
	}
	return out, nil
}

func (r *EventRepo) GetByID(ctx context.Context, id string) (domain.Event, error) {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	er, ok := r.s.events[id]
	if !ok {
		return domain.Event{}, domain.ErrNotFound("event")
	}
	return er.Event, nil
}

func (r *EventRepo) Create(ctx context.Context, e *domain.Event) error {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	if _, ok := r.s.schedules[e.ScheduleID]; !ok {
		return domain.NewError(domain.KindNotFound, "referenced record does not exist")
	}
	now := time.Now().UTC().Truncate(time.Second)
	e.CreatedAt, e.UpdatedAt = now, now
	if e.ID == "" {
		e.ID = domain.NewID()
	}
	if _, exists := r.s.events[e.ID]; exists {
		return domain.NewError(domain.KindConflict, "schedule does not exist")
	}
	r.s.seq++
	r.s.events[e.ID] = eventRecord{Event: *e, seq: r.s.seq}
	return nil
}

func (r *EventRepo) Update(ctx context.Context, e *domain.Event) error {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	er, ok := r.s.events[e.ID]
	if !ok {
		return domain.ErrNotFound("event")
	}
	now := time.Now().UTC().Truncate(time.Second)
	e.UpdatedAt = now
	er.Name = e.Name
	er.Description = e.Description
	er.Location = e.Location
	er.StartsAt = e.StartsAt
	er.AllDay = e.AllDay
	er.EndsAt = e.EndsAt
	er.UpdatedAt = now
	r.s.events[e.ID] = er
	return nil
}

func (r *EventRepo) Delete(ctx context.Context, id string) error {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	if _, ok := r.s.events[id]; !ok {
		return domain.ErrNotFound("event")
	}
	delete(r.s.events, id)
	for aid, ar := range r.s.attendees {
		if ar.EventID == id {
			delete(r.s.attendees, aid)
		}
	}
	return nil
}

func (r *AttendeeRepo) ListByEvent(ctx context.Context, eventID string) ([]domain.Attendee, error) {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	recs := make([]attendeeRecord, 0)
	for _, ar := range r.s.attendees {
		if ar.EventID == eventID {
			recs = append(recs, ar)
		}
	}
	slices.SortFunc(recs, func(a, b attendeeRecord) int {
		if c := a.CreatedAt.Compare(b.CreatedAt); c != 0 {
			return c
		}
		return cmp.Compare(a.seq, b.seq)
	})
	out := make([]domain.Attendee, len(recs))
	for i, rec := range recs {
		out[i] = rec.Attendee
	}
	return out, nil
}

func (r *AttendeeRepo) GetByID(ctx context.Context, id string) (domain.Attendee, error) {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	ar, ok := r.s.attendees[id]
	if !ok {
		return domain.Attendee{}, domain.ErrNotFound("attendee")
	}
	return ar.Attendee, nil
}

func (r *AttendeeRepo) FindByEventAndNormalizedName(ctx context.Context, eventID string, normalized string) (domain.Attendee, error) {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	for _, ar := range r.s.attendees {
		if ar.EventID == eventID && ar.NormalizedName == normalized {
			return ar.Attendee, nil
		}
	}
	return domain.Attendee{}, domain.ErrNotFound("attendee")
}

func (r *AttendeeRepo) Create(ctx context.Context, a *domain.Attendee) error {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	if _, ok := r.s.events[a.EventID]; !ok {
		return domain.NewError(domain.KindNotFound, "referenced record does not exist")
	}
	for _, ar := range r.s.attendees {
		if (ar.EventID == a.EventID && ar.NormalizedName == a.NormalizedName) || ar.ManageToken == a.ManageToken {
			return domain.NewError(domain.KindConflict, "this name is already taken for this event")
		}
	}
	now := time.Now().UTC().Truncate(time.Second)
	a.CreatedAt = now
	if a.ID == "" {
		a.ID = domain.NewID()
	}
	if _, exists := r.s.attendees[a.ID]; exists {
		return domain.NewError(domain.KindConflict, "this name is already taken for this event")
	}
	r.s.seq++
	r.s.attendees[a.ID] = attendeeRecord{Attendee: *a, seq: r.s.seq}
	return nil
}

func (r *AttendeeRepo) UpdateToken(ctx context.Context, id string, token string) error {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	ar, ok := r.s.attendees[id]
	if !ok {
		return domain.ErrNotFound("attendee")
	}
	for _, other := range r.s.attendees {
		if other.ID != id && other.ManageToken == token {
			return domain.NewError(domain.KindConflict, "token conflict")
		}
	}
	ar.ManageToken = token
	r.s.attendees[id] = ar
	return nil
}

func (r *AttendeeRepo) Delete(ctx context.Context, id string) error {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	if _, ok := r.s.attendees[id]; !ok {
		return domain.ErrNotFound("attendee")
	}
	delete(r.s.attendees, id)
	return nil
}

func (r *SettingsRepo) Get(ctx context.Context, key string) (string, error) {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	v, ok := r.s.settings[key]
	if !ok {
		return "", domain.ErrNotFound("setting")
	}
	return v, nil
}

func (r *SettingsRepo) Set(ctx context.Context, key, value string) error {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	r.s.settings[key] = value
	return nil
}

func (r *SettingsRepo) All(ctx context.Context) (map[string]string, error) {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	return maps.Clone(r.s.settings), nil
}
