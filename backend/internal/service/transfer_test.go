package service

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"shale/internal/db"
	"shale/internal/domain"
	"shale/internal/repository/sqlite"
)

type transferEnv struct {
	schedules   *ScheduleService
	events      *EventService
	settings    *SettingsService
	transfer    *TransferService
	attendeeSvc *AttendeeService
}

func newTransferEnv(t *testing.T) *transferEnv {
	t.Helper()
	conn, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { conn.Close() })
	if err := db.Migrate(context.Background(), conn); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	store := sqlite.New(conn)
	scheduleRepo := sqlite.NewScheduleRepo(store)
	eventRepo := sqlite.NewEventRepo(store)
	attendeeRepo := sqlite.NewAttendeeRepo(store)
	settingsRepo := sqlite.NewSettingsRepo(store)
	clock := func() time.Time { return time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC) }
	schedules := NewScheduleService(scheduleRepo, eventRepo, attendeeRepo, clock)
	events := NewEventService(scheduleRepo, eventRepo)
	settings := NewSettingsService(settingsRepo)
	transfer := NewTransferService(scheduleRepo, eventRepo, attendeeRepo, settings, store, clock)
	return &transferEnv{
		schedules:   schedules,
		events:      events,
		settings:    settings,
		transfer:    transfer,
		attendeeSvc: NewAttendeeService(scheduleRepo, eventRepo, attendeeRepo, clock),
	}
}

func seedFullState(t *testing.T, env *transferEnv) string {
	t.Helper()
	ctx := context.Background()
	sc, err := env.schedules.Create(ctx, ScheduleInput{Title: "Training week 42", Description: "week of trainings"})
	if err != nil {
		t.Fatalf("create schedule: %v", err)
	}
	if _, err := env.events.Create(ctx, sc.ID, EventInput{Name: "Mobility", Location: "Gym A", StartsAt: "2026-10-12T09:00"}); err != nil {
		t.Fatalf("create event: %v", err)
	}
	ev2, err := env.events.Create(ctx, sc.ID, EventInput{Name: "Strength", Location: "Gym B", StartsAt: "2026-10-13T10:30", Description: "bring shoes"})
	if err != nil {
		t.Fatalf("create event: %v", err)
	}

	if _, err := env.attendeeSvc.Attend(ctx, sc.ID, ev2.ID, "Alice"); err != nil {
		t.Fatalf("attend: %v", err)
	}
	if _, err := env.attendeeSvc.Attend(ctx, sc.ID, ev2.ID, "Bob"); err != nil {
		t.Fatalf("attend: %v", err)
	}
	if err := env.settings.SetDefaultLanguage(ctx, domain.LanguageDE); err != nil {
		t.Fatalf("set language: %v", err)
	}
	return sc.ID
}

func TestExportImportRoundTrip(t *testing.T) {
	env := newTransferEnv(t)
	ctx := context.Background()
	scheduleID := seedFullState(t, env)

	doc, err := env.transfer.Export(ctx)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("marshal export: %v", err)
	}

	if strings.Contains(string(raw), "manage_token") {
		t.Error("export contains manage_token key")
	}

	fresh := newTransferEnv(t)
	counts, err := fresh.transfer.Import(ctx, raw)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if counts.Schedules != 1 || counts.Events != 2 || counts.Attendees != 2 {
		t.Errorf("counts = %+v, want 1/2/2", counts)
	}

	ps, err := fresh.schedules.GetPublic(ctx, scheduleID)
	if err != nil {

		list, lerr := fresh.schedules.List(ctx)
		if lerr != nil || len(list) != 1 {
			t.Fatalf("list after import: %v", lerr)
		}
		ps, err = fresh.schedules.GetPublic(ctx, list[0].ID)
		if err != nil {
			t.Fatalf("GetPublic after import: %v", err)
		}
	}
	if ps.Schedule.Title != "Training week 42" || len(ps.Events) != 2 {
		t.Fatalf("unexpected schedule: %+v", ps)
	}
	var strength PublicEvent
	for _, pe := range ps.Events {
		if pe.Event.Name == "Strength" {
			strength = pe
		}
	}
	if len(strength.Attendees) != 2 {
		t.Errorf("attendees after import = %d, want 2", len(strength.Attendees))
	}

	view, err := fresh.settings.Get(ctx)
	if err != nil {
		t.Fatalf("settings after import: %v", err)
	}
	if view.DefaultLanguage != domain.LanguageDE {
		t.Errorf("default_language = %q, want de", view.DefaultLanguage)
	}
}

func TestImportValidation(t *testing.T) {
	env := newTransferEnv(t)
	ctx := context.Background()

	tests := []struct {
		name    string
		raw     string
		wantErr bool
		code    domain.Kind
	}{
		{
			name:    "not json",
			raw:     `nope`,
			wantErr: true,
			code:    domain.KindInvalid,
		},
		{
			name:    "wrong app",
			raw:     `{"app":"other","schema_version":2,"schedules":[]}`,
			wantErr: true,
			code:    domain.KindInvalid,
		},
		{
			name:    "unsupported schema version",
			raw:     `{"app":"shale","schema_version":99,"schedules":[]}`,
			wantErr: true,
			code:    domain.KindUnprocessable,
		},
		{
			name:    "invalid starts_at",
			raw:     `{"app":"shale","schema_version":2,"schedules":[{"title":"t","events":[{"name":"e","starts_at":"tomorrow","attendees":[]}]}]}`,
			wantErr: true,
			code:    domain.KindUnprocessable,
		},
		{
			name:    "valid minimal import",
			raw:     `{"app":"shale","schema_version":2,"schedules":[{"title":"t","events":[{"name":"e","starts_at":"2026-10-12T09:00","attendees":[{"name":"Alice"}]}]}]}`,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			counts, err := env.transfer.Import(ctx, []byte(tt.raw))
			if tt.wantErr != (err != nil) {
				t.Fatalf("Import() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				if k, ok := domain.KindOf(err); ok && k != tt.code {
					t.Errorf("error kind = %q, want %q", k, tt.code)
				}
				return
			}
			if counts.Events != 1 || counts.Attendees != 1 {
				t.Errorf("counts = %+v, want 1 event / 1 attendee", counts)
			}
		})
	}
}

func TestImportReplacesEverything(t *testing.T) {
	env := newTransferEnv(t)
	ctx := context.Background()
	seedFullState(t, env)

	raw := `{"app":"shale","schema_version":2,"settings":{"default_language":"en"},"schedules":[{"title":"Only one","events":[]}]}`
	if _, err := env.transfer.Import(ctx, []byte(raw)); err != nil {
		t.Fatalf("Import: %v", err)
	}
	list, err := env.schedules.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 1 || list[0].Title != "Only one" {
		t.Errorf("schedules after import = %+v, want only 'Only one'", list)
	}
}
