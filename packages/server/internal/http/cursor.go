package httpapi

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"
)

// cursorTimeLayout is the timestamp format embedded in a keyset-pagination
// cursor: RFC3339Nano, matching how created_at is stored/compared in SQLite.
const cursorTimeLayout = time.RFC3339Nano

// errMalformedCursor is returned by decodeCursor for any input that does not
// decode to "<RFC3339Nano timestamp>|<id>". The HTTP layer maps it to 400.
var errMalformedCursor = errors.New("cursor: malformed")

// defaultListLimit and maxListLimit bound every keyset-paginated list
// endpoint's ?limit query param: a non-positive value defaults to
// defaultListLimit, and anything above maxListLimit is capped.
const (
	defaultListLimit = 30
	maxListLimit     = 100
)

// clampListLimit applies the shared ?limit clamp for keyset-paginated list
// endpoints. The clamped value it returns is also the exact row count that
// means "there may be another page": a handler compares the number of rows
// fetched against this same clamped limit to decide whether to set
// X-Next-Cursor.
func clampListLimit(limit int) int {
	if limit <= 0 {
		return defaultListLimit
	}
	if limit > maxListLimit {
		return maxListLimit
	}
	return limit
}

// encodeCursor builds the opaque keyset-pagination cursor for a page boundary
// row: base64url("<created_at in RFC3339Nano>|<id>"). It is the exact
// contract every keyset-paginated list endpoint (runs, reviews) uses for its
// X-Next-Cursor header.
func encodeCursor(t time.Time, id string) string {
	raw := t.UTC().Format(cursorTimeLayout) + "|" + id
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

// decodeCursor reverses encodeCursor. A malformed or unparseable cursor
// returns errMalformedCursor (wrapped), which the HTTP layer maps to 400.
func decodeCursor(s string) (time.Time, string, error) {
	raw, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return time.Time{}, "", fmt.Errorf("%w: %v", errMalformedCursor, err)
	}
	ts, id, ok := strings.Cut(string(raw), "|")
	if !ok || ts == "" || id == "" {
		return time.Time{}, "", errMalformedCursor
	}
	t, err := time.Parse(cursorTimeLayout, ts)
	if err != nil {
		return time.Time{}, "", fmt.Errorf("%w: %v", errMalformedCursor, err)
	}
	return t, id, nil
}
