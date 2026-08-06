// Package openrouter reads OpenRouter's public model catalog. The catalog is a
// no-auth endpoint (GET https://openrouter.ai/api/v1/models), so this adapter
// only needs a plain HTTP client. A small TTL cache sits in front of it so
// repeated form opens / keystrokes in the UI don't hammer OpenRouter.
package openrouter

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// modelsURL is OpenRouter's public, no-auth model catalog endpoint.
const modelsURL = "https://openrouter.ai/api/v1/models"

// maxCatalogBytes bounds the response body read. The catalog is large-ish
// (hundreds of models), so this is generous while still capping a runaway body.
const maxCatalogBytes = 8 << 20 // 8 MiB

// Model is the subset of an OpenRouter catalog entry the UI needs.
type Model struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	ContextLength int    `json:"context_length"`
}

// Client fetches the OpenRouter model catalog over HTTP (no auth).
type Client struct {
	http *http.Client
	url  string
}

// NewClient builds a catalog client with a ~10s timeout.
func NewClient() *Client {
	return &Client{
		http: &http.Client{Timeout: 10 * time.Second},
		url:  modelsURL,
	}
}

// ListModels GETs the catalog and decodes its data[] entries.
func (c *Client) ListModels(ctx context.Context) ([]Model, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url, nil)
	if err != nil {
		return nil, fmt.Errorf("openrouter: build request: %w", err)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("openrouter: fetch catalog: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("openrouter: catalog returned status %d", resp.StatusCode)
	}
	var body struct {
		Data []Model `json:"data"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxCatalogBytes)).Decode(&body); err != nil {
		return nil, fmt.Errorf("openrouter: decode catalog: %w", err)
	}
	return body.Data, nil
}

// CachedClient wraps a fetch with a mutex-guarded TTL cache. On a fetch error it
// serves the last cached data if any, otherwise returns the error. This is
// server-side code, so time.Now() is fine here.
type CachedClient struct {
	fetch func(ctx context.Context) ([]Model, error)
	ttl   time.Duration

	mu        sync.Mutex
	data      []Model
	fetchedAt time.Time
	cached    bool
}

// NewCachedClient builds a CachedClient over the live OpenRouter catalog with
// the given TTL (e.g. ~1h).
func NewCachedClient(ttl time.Duration) *CachedClient {
	return &CachedClient{fetch: NewClient().ListModels, ttl: ttl}
}

// ListModels returns cached models when fresh, otherwise refetches. A refetch
// error falls back to stale cached data when available.
func (c *CachedClient) ListModels(ctx context.Context) ([]Model, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cached && time.Since(c.fetchedAt) < c.ttl {
		return c.data, nil
	}

	data, err := c.fetch(ctx)
	if err != nil {
		if c.cached {
			return c.data, nil // serve stale data on error
		}
		return nil, err
	}
	c.data = data
	c.fetchedAt = time.Now()
	c.cached = true
	return data, nil
}
