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

// Profile captures a writing voice. Samples are raw writing pasted by the user;
// StyleGuide is an LLM-distilled cache filled by a later slice (empty for now).
// Samples are not secret, so they are stored in the clear (unlike API keys).
type Profile struct {
	ID         string
	Name       string
	Language   string   // free text, e.g. "es-AR", "en"
	Formality  string   // free text, e.g. casual|neutral|formal
	Emojis     bool     // whether the voice uses emojis
	Samples    []string // raw writing samples the user pasted
	StyleGuide string   // LLM-distilled cache; empty until distillation is added
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// Repository persists humanization profiles.
type Repository interface {
	Create(ctx context.Context, p Profile) error
	// Save updates the mutable fields of an existing profile.
	Save(ctx context.Context, p Profile) error
	Get(ctx context.Context, id string) (Profile, error)
	List(ctx context.Context) ([]Profile, error)
	Delete(ctx context.Context, id string) error
}
