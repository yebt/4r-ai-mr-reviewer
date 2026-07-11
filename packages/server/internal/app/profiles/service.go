// Package profiles coordinates humanization profile persistence. Profiles hold
// no secrets, so (unlike providers) the service has no secret-store dependency.
package profiles

import (
	"context"
	"fmt"
	"time"

	"github.com/webcloster-dev/ai-reviewer/internal/domain/profile"
	"github.com/webcloster-dev/ai-reviewer/internal/id"
)

// Service manages humanization profiles.
type Service struct {
	repo profile.Repository
}

// NewService wires the profile service.
func NewService(repo profile.Repository) *Service {
	return &Service{repo: repo}
}

// AddInput describes a new profile. StyleGuide is server-managed and stays empty
// until the distillation slice fills it, so it is not accepted here.
type AddInput struct {
	Name      string
	Language  string
	Formality string
	Emojis    bool
	Samples   []string
}

// Add records a new profile.
func (s *Service) Add(ctx context.Context, in AddInput) (profile.Profile, error) {
	if in.Name == "" {
		return profile.Profile{}, fmt.Errorf("profiles: name is required")
	}

	now := time.Now().UTC()
	p := profile.Profile{
		ID:        id.New(),
		Name:      in.Name,
		Language:  in.Language,
		Formality: in.Formality,
		Emojis:    in.Emojis,
		Samples:   in.Samples,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.repo.Create(ctx, p); err != nil {
		return profile.Profile{}, err
	}
	return p, nil
}

// List returns all profiles.
func (s *Service) List(ctx context.Context) ([]profile.Profile, error) {
	return s.repo.List(ctx)
}

// Get returns one profile.
func (s *Service) Get(ctx context.Context, id string) (profile.Profile, error) {
	return s.repo.Get(ctx, id)
}

// UpdateInput describes an edit to a profile. StyleGuide is server-managed and
// is never overwritten here.
type UpdateInput struct {
	Name      string
	Language  string
	Formality string
	Emojis    bool
	Samples   []string
}

// Update edits a profile's fields.
func (s *Service) Update(ctx context.Context, id string, in UpdateInput) (profile.Profile, error) {
	if in.Name == "" {
		return profile.Profile{}, fmt.Errorf("profiles: name is required")
	}

	p, err := s.repo.Get(ctx, id)
	if err != nil {
		return profile.Profile{}, err
	}
	p.Name = in.Name
	p.Language = in.Language
	p.Formality = in.Formality
	p.Emojis = in.Emojis
	p.Samples = in.Samples
	p.UpdatedAt = time.Now().UTC()

	if err := s.repo.Save(ctx, p); err != nil {
		return profile.Profile{}, err
	}
	return p, nil
}

// Delete removes a profile.
func (s *Service) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}
