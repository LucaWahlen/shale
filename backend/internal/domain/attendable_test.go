package domain

import (
	"testing"
	"time"
)

func at(y int, mo time.Month, d, h, mi int) time.Time {
	return time.Date(y, mo, d, h, mi, 0, 0, time.UTC)
}

func TestAttendable(t *testing.T) {
	tests := []struct {
		name     string
		startsAt string
		now      time.Time
		want     bool
	}{
		{name: "future", startsAt: "2026-09-20T09:00", now: at(2026, 9, 16, 12, 0), want: true},
		{name: "today later", startsAt: "2026-09-16T18:00", now: at(2026, 9, 16, 12, 0), want: true},
		{name: "today earlier", startsAt: "2026-09-16T08:00", now: at(2026, 9, 16, 12, 0), want: true},
		{name: "today exact", startsAt: "2026-09-16T12:00", now: at(2026, 9, 16, 12, 0), want: true},
		{name: "yesterday", startsAt: "2026-09-15T18:00", now: at(2026, 9, 16, 12, 0), want: false},
		{name: "last week", startsAt: "2026-09-09T09:00", now: at(2026, 9, 16, 12, 0), want: false},
		{name: "year boundary still same day", startsAt: "2026-01-01T00:00", now: at(2026, 1, 1, 23, 59), want: true},
		{name: "invalid format", startsAt: "not-a-date", now: at(2026, 9, 16, 12, 0), want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Attendable(tt.startsAt, tt.now); got != tt.want {
				t.Errorf("Attendable(%q, %v) = %v, want %v", tt.startsAt, tt.now, got, tt.want)
			}
		})
	}
}

func TestStartsAtFromParts(t *testing.T) {
	got, err := StartsAtFromParts("2026-10-12", "09:05")
	if err != nil {
		t.Fatalf("StartsAtFromParts() error = %v", err)
	}
	if got != "2026-10-12T09:05" {
		t.Errorf("StartsAtFromParts() = %q, want 2026-10-12T09:05", got)
	}
	if _, err := StartsAtFromParts("12.10.2026", "09:00"); err == nil {
		t.Error("expected error for bad date")
	}
	if _, err := StartsAtFromParts("2026-10-12", "25:00"); err == nil {
		t.Error("expected error for bad time")
	}
}

func TestDateWithOffset(t *testing.T) {
	tests := []struct {
		in     string
		offset int
		want   string
	}{
		{in: "2026-10-12", offset: 0, want: "2026-10-12"},
		{in: "2026-10-12", offset: 3, want: "2026-10-15"},
		{in: "2026-10-31", offset: 1, want: "2026-11-01"},
		{in: "2026-12-31", offset: 1, want: "2027-01-01"},
		{in: "2026-10-12", offset: -12, want: "2026-09-30"},
	}
	for _, tt := range tests {
		got, err := DateWithOffset(tt.in, tt.offset)
		if err != nil {
			t.Fatalf("DateWithOffset(%q, %d) error = %v", tt.in, tt.offset, err)
		}
		if got != tt.want {
			t.Errorf("DateWithOffset(%q, %d) = %q, want %q", tt.in, tt.offset, got, tt.want)
		}
	}
}

func TestDayOffset(t *testing.T) {
	got, ok := DayOffset("2026-10-12", "2026-10-16")
	if !ok || got != 4 {
		t.Errorf("DayOffset = %d, %v; want 4, true", got, ok)
	}
	got, ok = DayOffset("2026-10-16", "2026-10-12")
	if !ok || got != -4 {
		t.Errorf("DayOffset = %d, %v; want -4, true", got, ok)
	}
}

func TestDateAndTimeOfStartsAt(t *testing.T) {
	d, ok := DateOfStartsAt("2026-10-12T09:05")
	if !ok || d != "2026-10-12" {
		t.Errorf("DateOfStartsAt = %q, %v", d, ok)
	}
	tm, ok := TimeOfStartsAt("2026-10-12T09:05")
	if !ok || tm != "09:05" {
		t.Errorf("TimeOfStartsAt = %q, %v", tm, ok)
	}
}

func TestValidTimeOfDay(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{in: "00:00", want: true},
		{in: "23:59", want: true},
		{in: "9:00", want: false},
		{in: "24:00", want: false},
		{in: "12:60", want: false},
		{in: "", want: false},
	}
	for _, tt := range tests {
		if got := ValidTimeOfDay(tt.in); got != tt.want {
			t.Errorf("ValidTimeOfDay(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}
