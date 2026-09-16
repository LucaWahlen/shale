package domain

import (
	"strconv"
	"strings"
	"time"
)

const (
	StartsAtFormat  = "2006-01-02T15:04"
	DateFormat      = "2006-01-02"
	TimeOfDayFormat = "15:04"
)

func ParseStartsAt(s string) (time.Time, bool) {
	t, err := time.Parse(StartsAtFormat, s)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

func ValidStartsAt(s string) bool {
	_, ok := ParseStartsAt(s)
	return ok
}

func ValidTimeOfDay(s string) bool {
	if len(s) != 5 || s[2] != ':' {
		return false
	}
	h := int(s[0]-'0')*10 + int(s[1]-'0')
	m := int(s[3]-'0')*10 + int(s[4]-'0')
	for _, i := range []int{0, 1, 3, 4} {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return h <= 23 && m <= 59
}

func ValidEndsAt(startsAt, endsAt string) bool {
	st, ok := ParseStartsAt(startsAt)
	if !ok {
		return false
	}
	et, ok := ParseStartsAt(endsAt)
	if !ok {
		return false
	}
	return et.After(st)
}

func NormalizeAllDayStartsAt(startsAt string) (string, bool) {
	st, ok := ParseStartsAt(startsAt)
	if !ok {
		return "", false
	}
	return time.Date(st.Year(), st.Month(), st.Day(), 0, 0, 0, 0, time.UTC).Format(StartsAtFormat), true
}

func Attendable(startsAt string, now time.Time) bool {
	st, ok := ParseStartsAt(startsAt)
	if !ok {
		return false
	}
	naiveNow := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), 0, 0, time.UTC)
	if !st.Before(naiveNow) {
		return true
	}
	return sameDate(st, naiveNow)
}

func sameDate(a, b time.Time) bool {
	return a.Year() == b.Year() && a.Month() == b.Month() && a.Day() == b.Day()
}

func StartsAtFromParts(date, timeOfDay string) (string, error) {
	d, err := time.Parse(DateFormat, date)
	if err != nil {
		return "", NewError(KindInvalid, "date must be in YYYY-MM-DD form")
	}
	if !ValidTimeOfDay(timeOfDay) {
		return "", NewError(KindInvalid, "time must be in HH:MM form")
	}
	tod := strings.Split(timeOfDay, ":")
	h, _ := strconv.Atoi(tod[0])
	mi, _ := strconv.Atoi(tod[1])
	full := time.Date(d.Year(), d.Month(), d.Day(), h, mi, 0, 0, time.UTC)
	return full.Format(StartsAtFormat), nil
}

func DateWithOffset(date string, n int) (string, error) {
	d, err := time.Parse(DateFormat, date)
	if err != nil {
		return "", NewError(KindInvalid, "date must be in YYYY-MM-DD form")
	}
	return d.AddDate(0, 0, n).Format(DateFormat), nil
}

func DateOfStartsAt(startsAt string) (string, bool) {
	t, ok := ParseStartsAt(startsAt)
	if !ok {
		return "", false
	}
	return t.Format(DateFormat), true
}

func TimeOfStartsAt(startsAt string) (string, bool) {
	t, ok := ParseStartsAt(startsAt)
	if !ok {
		return "", false
	}
	return t.Format(TimeOfDayFormat), true
}

func DayOffset(from, to string) (int, bool) {
	f, err := time.Parse(DateFormat, from)
	if err != nil {
		return 0, false
	}
	t, err := time.Parse(DateFormat, to)
	if err != nil {
		return 0, false
	}
	return int(t.Sub(f).Hours() / 24), true
}
