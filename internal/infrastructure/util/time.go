package util

import "time"

// ParseTime parses an RFC 3339 timestamp string, falling back to a loose layout.
func ParseTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t, _ = time.Parse("2006-01-02 15:04:05", s)
	}
	return t
}
