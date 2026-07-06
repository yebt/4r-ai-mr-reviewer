package sqlite

import "time"

// timeLayout is how timestamps are stored in TEXT columns (UTC, RFC3339Nano).
const timeLayout = time.RFC3339Nano

func formatTime(t time.Time) string { return t.UTC().Format(timeLayout) }

func parseTime(s string) (time.Time, error) { return time.Parse(timeLayout, s) }
