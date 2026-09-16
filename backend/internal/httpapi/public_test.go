package httpapi

import (
	"net/http"
	"testing"
	"time"
)

func TestPublicSettings(t *testing.T) {
	env := newTestEnv(t, time.Unix(1_800_000_000, 0))
	res, out := env.do(http.MethodGet, "/api/v1/settings", nil, nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", res.StatusCode)
	}
	if out["default_language"] != "en" {
		t.Errorf("default_language = %v, want en", out["default_language"])
	}
}

func TestPublicScheduleNotFound(t *testing.T) {
	env := newTestEnv(t, time.Unix(1_800_000_000, 0))
	res, out := env.do(http.MethodGet, "/api/v1/schedules/00000000-0000-0000-0000-000000000000", nil, nil)
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", res.StatusCode)
	}
	if out["error"] == nil {
		t.Errorf("expected error envelope, got %v", out)
	}
}

func TestAttendAndRevertFlow(t *testing.T) {
	now := time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)
	env := newTestEnv(t, now)
	scheduleID, eventID := env.seedSchedule("2026-09-20T09:00")
	attendPath := "/api/v1/schedules/" + scheduleID + "/events/" + eventID + "/attend"

	res, out := env.do(http.MethodPost, attendPath, map[string]string{"name": "Alice"}, nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("attend status = %d, want 200; body %v", res.StatusCode, out)
	}
	if out["created"] != true {
		t.Errorf("created = %v, want true", out["created"])
	}
	att := out["attendee"].(map[string]any)
	attendeeID := att["id"].(string)
	token := out["manage_token"].(string)
	if token == "" {
		t.Fatal("manage_token empty")
	}

	res, out = env.do(http.MethodPost, attendPath, map[string]string{"name": "  aLiCe "}, nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("re-attend status = %d, want 200", res.StatusCode)
	}
	if out["created"] != false {
		t.Errorf("created = %v, want false", out["created"])
	}
	att2 := out["attendee"].(map[string]any)
	if att2["id"].(string) != attendeeID {
		t.Errorf("re-attend resolved to attendee %v, want %s", att2["id"], attendeeID)
	}
	token = out["manage_token"].(string)

	res, sched := env.do(http.MethodGet, "/api/v1/schedules/"+scheduleID, nil, nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("schedule status = %d", res.StatusCode)
	}
	events := sched["events"].([]any)
	ev := events[0].(map[string]any)
	if len(ev["attendees"].([]any)) != 1 {
		t.Errorf("attendees = %v, want exactly one entry", ev["attendees"])
	}

	res, _ = env.do(http.MethodDelete, attendPath, map[string]any{
		"attendee_id": attendeeID, "manage_token": "deadbeefdeadbeefdeadbeefdeadbeef",
	}, nil)
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("revert wrong token status = %d, want 403", res.StatusCode)
	}

	res, _ = env.do(http.MethodDelete, attendPath, map[string]any{
		"attendee_id": attendeeID, "manage_token": token,
	}, nil)
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("revert status = %d, want 204", res.StatusCode)
	}

	_, sched = env.do(http.MethodGet, "/api/v1/schedules/"+scheduleID, nil, nil)
	events = sched["events"].([]any)
	ev = events[0].(map[string]any)
	if len(ev["attendees"].([]any)) != 0 {
		t.Errorf("attendees after revert = %v, want empty", ev["attendees"])
	}
}

func TestAttendPastEvent(t *testing.T) {
	now := time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)
	env := newTestEnv(t, now)
	scheduleID, eventID := env.seedSchedule("2026-09-15T09:00")
	attendPath := "/api/v1/schedules/" + scheduleID + "/events/" + eventID + "/attend"

	res, out := env.do(http.MethodPost, attendPath, map[string]string{"name": "Alice"}, nil)
	if res.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("attend past status = %d, want 422; body %v", res.StatusCode, out)
	}
}

func TestRevertPastEvent(t *testing.T) {
	now := time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)
	env := newTestEnv(t, now)

	scheduleID, eventID := env.seedSchedule("2026-09-15T09:00")
	res, out := env.do(http.MethodPost, "/api/v1/admin/events/"+eventID+"/attendees", map[string]string{"name": "Alice"}, env.login())
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("admin add status = %d; body %v", res.StatusCode, out)
	}
	attendeeID := out["id"].(string)

	res, _ = env.do(http.MethodDelete, "/api/v1/schedules/"+scheduleID+"/events/"+eventID+"/attend", map[string]any{
		"attendee_id": attendeeID, "manage_token": "whatever",
	}, nil)
	if res.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("revert past status = %d, want 422", res.StatusCode)
	}
}

func TestAttendValidation(t *testing.T) {
	now := time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)
	env := newTestEnv(t, now)
	scheduleID, eventID := env.seedSchedule("2026-09-20T09:00")
	attendPath := "/api/v1/schedules/" + scheduleID + "/events/" + eventID + "/attend"

	tests := []struct {
		name   string
		body   map[string]string
		status int
	}{
		{name: "empty name", body: map[string]string{"name": "   "}, status: 400},
		{name: "too long", body: map[string]string{"name": string(makeName(81))}, status: 400},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, _ := env.do(http.MethodPost, attendPath, tt.body, nil)
			if res.StatusCode != tt.status {
				t.Errorf("status = %d, want %d", res.StatusCode, tt.status)
			}
		})
	}
}

func TestUnknownAPIRoute(t *testing.T) {
	env := newTestEnv(t, time.Unix(1_800_000_000, 0))
	res, out := env.do(http.MethodGet, "/api/v1/unknown", nil, nil)
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", res.StatusCode)
	}
	if out["error"] == nil {
		t.Error("expected JSON error envelope")
	}
}

func TestDuplicateSchedule(t *testing.T) {
	now := time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)
	env := newTestEnv(t, now)
	scheduleID, eventID := env.seedSchedule("2026-09-20T09:00")

	res, _ := env.do(http.MethodPost, "/api/v1/schedules/"+scheduleID+"/events/"+eventID+"/attend", map[string]string{"name": "Alice"}, nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("attend status = %d", res.StatusCode)
	}

	res, out := env.do(http.MethodPost, "/api/v1/admin/schedules/"+scheduleID+"/duplicate", nil, env.login())
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("duplicate status = %d; body %v", res.StatusCode, out)
	}
	newID := out["id"].(string)
	if out["title"] != "Test Schedule (copy)" {
		t.Errorf("title = %v, want 'Test Schedule (copy)'", out["title"])
	}
	if newID == scheduleID {
		t.Fatal("duplicate got the same ID")
	}

	_, detail := env.do(http.MethodGet, "/api/v1/admin/schedules/"+newID, nil, env.login())
	events := detail["events"].([]any)
	if len(events) != 1 {
		t.Fatalf("duplicated events = %d, want 1", len(events))
	}
	if len(events[0].(map[string]any)["attendees"].([]any)) != 0 {
		t.Error("attendees must not be duplicated")
	}
}

func makeName(n int) []byte {
	out := make([]byte, n)
	for i := range out {
		out[i] = 'a'
	}
	return out
}
