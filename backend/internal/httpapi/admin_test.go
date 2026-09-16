package httpapi

import (
	"net/http"
	"testing"
	"time"
)

func TestAdminRequiresAuth(t *testing.T) {
	env := newTestEnv(t, time.Unix(1_800_000_000, 0))
	res, _ := env.do(http.MethodGet, "/api/v1/admin/schedules", nil, nil)
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", res.StatusCode)
	}
}

func TestAdminLoginLogout(t *testing.T) {
	env := newTestEnv(t, time.Unix(1_800_000_000, 0))

	res, _ := env.do(http.MethodPost, "/api/v1/admin/login", map[string]string{"password": "wrong"}, nil)
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("wrong password status = %d, want 401", res.StatusCode)
	}

	cookie := env.login()

	res, _ = env.do(http.MethodGet, "/api/v1/admin/schedules", nil, cookie)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("authed list status = %d, want 200", res.StatusCode)
	}

	res, _ = env.do(http.MethodPost, "/api/v1/admin/logout", nil, cookie)
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("logout status = %d, want 204", res.StatusCode)
	}
	var cleared *http.Cookie
	for _, c := range res.Cookies() {
		if c.Name == SessionCookieName {
			cleared = c
		}
	}
	if cleared == nil || cleared.MaxAge >= 0 {
		t.Error("logout did not clear the session cookie")
	}
}

func TestAdminRateLimit(t *testing.T) {
	env := newTestEnv(t, time.Unix(1_800_000_000, 0))
	for i := 0; i < 5; i++ {
		res, _ := env.do(http.MethodPost, "/api/v1/admin/login", map[string]string{"password": "wrong"}, nil)
		if res.StatusCode != http.StatusUnauthorized {
			t.Fatalf("attempt %d status = %d, want 401", i+1, res.StatusCode)
		}
	}

	res, _ := env.do(http.MethodPost, "/api/v1/admin/login", map[string]string{"password": "test-password"}, nil)
	if res.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("6th attempt status = %d, want 429", res.StatusCode)
	}
}

func TestAdminListIncludesEventsAndAttendees(t *testing.T) {
	env := newTestEnv(t, time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC))
	cookie := env.login()
	_, eventID := env.seedSchedule("2026-09-20T09:00")

	res, _ := env.do(http.MethodPost, "/api/v1/admin/events/"+eventID+"/attendees", map[string]string{"name": "Bob"}, cookie)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("add attendee status = %d", res.StatusCode)
	}

	var list []map[string]any
	res = env.doInto(http.MethodGet, "/api/v1/admin/schedules", nil, cookie, &list)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("list status = %d", res.StatusCode)
	}
	if len(list) != 1 {
		t.Fatalf("list length = %d, want 1", len(list))
	}
	s := list[0]
	events, ok := s["events"].([]any)
	if !ok || len(events) != 1 {
		t.Fatalf("expected 1 embedded event in list response, got %v", s["events"])
	}
	ev := events[0].(map[string]any)
	attendees, ok := ev["attendees"].([]any)
	if !ok || len(attendees) != 1 {
		t.Fatalf("expected 1 embedded attendee in list response, got %v", ev["attendees"])
	}
	if attendees[0].(map[string]any)["name"] != "Bob" {
		t.Errorf("attendee name = %v, want Bob", attendees[0].(map[string]any)["name"])
	}
	if s["event_count"] != float64(1) {
		t.Errorf("event_count = %v, want 1", s["event_count"])
	}
	if s["first_starts_at"] != "2026-09-20T09:00" {
		t.Errorf("first_starts_at = %v, want 2026-09-20T09:00", s["first_starts_at"])
	}
	if s["last_starts_at"] != "2026-09-20T09:00" {
		t.Errorf("last_starts_at = %v, want 2026-09-20T09:00", s["last_starts_at"])
	}
}

func TestAdminScheduleCRUD(t *testing.T) {
	env := newTestEnv(t, time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC))
	cookie := env.login()

	res, out := env.do(http.MethodPost, "/api/v1/admin/schedules", map[string]string{
		"title": "My Week", "description": "desc",
	}, cookie)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("create status = %d; %v", res.StatusCode, out)
	}
	id := out["id"].(string)

	res, out = env.do(http.MethodGet, "/api/v1/admin/schedules/"+id, nil, cookie)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("get status = %d", res.StatusCode)
	}
	if out["events"] == nil {
		t.Error("expected events array in admin schedule detail")
	}

	res, out = env.do(http.MethodPut, "/api/v1/admin/schedules/"+id, map[string]string{
		"title": "Renamed", "description": "",
	}, cookie)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("update status = %d", res.StatusCode)
	}
	if out["title"] != "Renamed" {
		t.Errorf("updated title = %v, want Renamed", out["title"])
	}

	res, _ = env.do(http.MethodDelete, "/api/v1/admin/schedules/"+id, nil, cookie)
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("delete status = %d, want 204", res.StatusCode)
	}

	res, _ = env.do(http.MethodGet, "/api/v1/admin/schedules/"+id, nil, cookie)
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("get after delete status = %d, want 404", res.StatusCode)
	}
}

func TestAdminEventAndAttendeeManagement(t *testing.T) {
	env := newTestEnv(t, time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC))
	cookie := env.login()
	scheduleID, eventID := env.seedSchedule("2026-09-20T09:00")

	res, out := env.do(http.MethodPatch, "/api/v1/admin/events/"+eventID, map[string]any{
		"location": "Gym A",
	}, cookie)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("patch status = %d; body %v", res.StatusCode, out)
	}
	if out["location"] != "Gym A" {
		t.Errorf("location = %v, want Gym A", out["location"])
	}

	res, out = env.do(http.MethodPost, "/api/v1/admin/events/"+eventID+"/attendees", map[string]string{"name": "Bob"}, cookie)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("add attendee status = %d; body %v", res.StatusCode, out)
	}
	attendeeID := out["id"].(string)

	res, out = env.do(http.MethodPost, "/api/v1/admin/events/"+eventID+"/attendees", map[string]string{"name": "bob "}, cookie)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("second add status = %d", res.StatusCode)
	}
	if out["id"].(string) != attendeeID {
		t.Errorf("second add id = %v, want %s", out["id"], attendeeID)
	}

	res, _ = env.do(http.MethodDelete, "/api/v1/admin/attendees/"+attendeeID, nil, cookie)
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("remove attendee status = %d, want 204", res.StatusCode)
	}

	res, _ = env.do(http.MethodDelete, "/api/v1/admin/events/"+eventID, nil, cookie)
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("delete event status = %d, want 204", res.StatusCode)
	}

	_, out = env.do(http.MethodGet, "/api/v1/admin/schedules/"+scheduleID, nil, cookie)
	if len(out["events"].([]any)) != 0 {
		t.Error("events not empty after delete")
	}
}

func TestAdminEventTimespan(t *testing.T) {
	env := newTestEnv(t, time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC))
	cookie := env.login()
	scheduleID, _ := env.seedSchedule("2026-09-20T09:00")

	create := func(body map[string]any) (*http.Response, map[string]any) {
		return env.do(http.MethodPost, "/api/v1/admin/schedules/"+scheduleID+"/events", body, cookie)
	}

	res, out := create(map[string]any{"name": "Full Day", "starts_at": "2026-09-22T17:30", "all_day": true})
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("create all-day status = %d; body %v", res.StatusCode, out)
	}
	if out["all_day"] != true || out["starts_at"] != "2026-09-22T00:00" || out["ends_at"] != "" {
		t.Errorf("all-day event = %v/%v/%v, want true/2026-09-22T00:00/''", out["all_day"], out["starts_at"], out["ends_at"])
	}

	res, out = create(map[string]any{"name": "Workshop", "starts_at": "2026-09-23T09:00", "ends_at": "2026-09-23T12:30"})
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("create timespan status = %d; body %v", res.StatusCode, out)
	}
	if out["ends_at"] != "2026-09-23T12:30" {
		t.Errorf("ends_at = %v, want 2026-09-23T12:30", out["ends_at"])
	}

	res, _ = create(map[string]any{"name": "Bad", "starts_at": "2026-09-23T09:00", "ends_at": "2026-09-23T08:00"})
	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("end before start status = %d, want 400", res.StatusCode)
	}

	res, _ = create(map[string]any{"name": "Bad", "starts_at": "2026-09-23T09:00", "all_day": true, "ends_at": "2026-09-23T10:00"})
	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("all-day with end status = %d, want 400", res.StatusCode)
	}

	eventID := out["id"].(string)
	res, _ = env.do(http.MethodPatch, "/api/v1/admin/events/"+eventID, map[string]any{"all_day": true}, cookie)
	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("patch all-day with existing ends_at status = %d, want 400", res.StatusCode)
	}
	res, out = env.do(http.MethodPatch, "/api/v1/admin/events/"+eventID, map[string]any{"all_day": true, "ends_at": ""}, cookie)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("patch all-day status = %d; body %v", res.StatusCode, out)
	}
	if out["starts_at"] != "2026-09-23T00:00" || out["all_day"] != true {
		t.Errorf("patched all-day = %v/%v, want true/2026-09-23T00:00", out["all_day"], out["starts_at"])
	}
}

func TestAdminSettings(t *testing.T) {
	env := newTestEnv(t, time.Unix(1_800_000_000, 0))
	cookie := env.login()

	res, out := env.do(http.MethodPut, "/api/v1/admin/settings", map[string]string{"default_language": "de"}, cookie)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("put settings status = %d; body %v", res.StatusCode, out)
	}

	_, out = env.do(http.MethodGet, "/api/v1/settings", nil, nil)
	if out["default_language"] != "de" {
		t.Errorf("default_language = %v, want de", out["default_language"])
	}
	if out["app_name"] != "shale" {
		t.Errorf("app_name default = %v, want shale", out["app_name"])
	}

	res, _ = env.do(http.MethodPut, "/api/v1/admin/settings", map[string]string{"default_language": "fr"}, cookie)
	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("invalid language status = %d, want 400", res.StatusCode)
	}
}

func TestAdminSettingsAppName(t *testing.T) {
	env := newTestEnv(t, time.Unix(1_800_000_000, 0))
	cookie := env.login()

	res, out := env.do(http.MethodPut, "/api/v1/admin/settings", map[string]any{
		"default_language": "en",
		"app_name":         "  Fußball Club  ",
	}, cookie)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("put settings status = %d; body %v", res.StatusCode, out)
	}
	if out["app_name"] != "Fußball Club" {
		t.Errorf("app_name = %v, want trimmed 'Fußball Club'", out["app_name"])
	}

	_, out = env.do(http.MethodGet, "/api/v1/admin/settings", nil, cookie)
	if out["app_name"] != "Fußball Club" {
		t.Errorf("admin app_name = %v, want 'Fußball Club'", out["app_name"])
	}
	_, out = env.do(http.MethodGet, "/api/v1/settings", nil, nil)
	if out["app_name"] != "Fußball Club" {
		t.Errorf("public app_name = %v, want 'Fußball Club'", out["app_name"])
	}

	res, out = env.do(http.MethodGet, "/api/v1/admin/export", nil, cookie)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("export status = %d", res.StatusCode)
	}
	settings, _ := out["settings"].(map[string]any)
	if settings["app_name"] != "Fußball Club" {
		t.Errorf("export settings.app_name = %v, want 'Fußball Club'", settings["app_name"])
	}

	res, _ = env.do(http.MethodPut, "/api/v1/admin/settings", map[string]any{
		"default_language": "en",
		"app_name":         "   ",
	}, cookie)
	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("empty app_name status = %d, want 400", res.StatusCode)
	}
}
