// Package profile models a humanization profile: the user's writing voice used
// later to rephrase review comments in their own style.
package profile

import (
	"context"
	"errors"
	"time"
)

// ErrNotFound is returned when a profile does not exist.
var ErrNotFound = errors.New("profile: not found")

// Style-guide distillation status. It tracks the async LLM distillation of the
// user's writing samples into a reusable StyleGuide.
const (
	// StyleStatusNone means no distillation applies (no samples / never run).
	StyleStatusNone = ""
	// StyleStatusPending means distillation was triggered and is in flight.
	StyleStatusPending = "pending"
	// StyleStatusReady means the StyleGuide holds a distilled result.
	StyleStatusReady = "ready"
	// StyleStatusError means the last distillation attempt failed; see
	// StyleGuideError for the reason.
	StyleStatusError = "error"
)

// Profile captures a writing voice. Samples are raw writing pasted by the user;
// StyleGuide is an LLM-distilled cache produced asynchronously from the samples.
// Samples are not secret, so they are stored in the clear (unlike API keys).
type Profile struct {
	ID         string
	Name       string
	Language   string   // free text, e.g. "es-AR", "en"
	Formality  string   // free text, e.g. casual|neutral|formal
	Emojis     bool     // whether the voice uses emojis
	Samples    []string // raw writing samples the user pasted
	StyleGuide string   // LLM-distilled cache; server-managed, filled async
	// StyleGuideStatus is one of the StyleStatus* values; server-managed.
	StyleGuideStatus string
	// StyleGuideError holds the last distillation failure message (if any).
	StyleGuideError string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// Repository persists humanization profiles.
type Repository interface {
	Create(ctx context.Context, p Profile) error
	// Save updates the mutable fields of an existing profile. It MUST NOT touch
	// the server-managed style-guide columns; use SetStyleGuide for those.
	Save(ctx context.Context, p Profile) error
	Get(ctx context.Context, id string) (Profile, error)
	List(ctx context.Context) ([]Profile, error)
	Delete(ctx context.Context, id string) error
	// SetStyleGuide records the distillation outcome (guide, status, error) for a
	// profile and bumps updated_at. Returns ErrNotFound when id is unknown.
	SetStyleGuide(ctx context.Context, id, guide, status, errMsg string) error
	// ListByStyleStatus returns profiles whose style-guide status equals status
	// (used by startup recovery to re-trigger pending distillations).
	ListByStyleStatus(ctx context.Context, status string) ([]Profile, error)
}
