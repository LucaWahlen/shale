package sqlite

import "time"

func timeNow() time.Time { return time.Now().UTC() }

func formatTime(t time.Time) string { return t.Format(time.RFC3339) }

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
