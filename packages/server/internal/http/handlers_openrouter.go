package httpapi

import (
	"context"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/webcloster-dev/ai-reviewer/internal/adapters/openrouter"
)

// openRouterCacheTTL is how long the OpenRouter model catalog is cached in front
// of the upstream endpoint so repeated form opens don't hammer OpenRouter.
const openRouterCacheTTL = time.Hour

// openRouterLister fetches the OpenRouter model catalog. The concrete
// implementation is a TTL-cached HTTP client; tests inject a stub.
type openRouterLister interface {
	ListModels(ctx context.Context) ([]openrouter.Model, error)
}

// openRouterModelDTO is the wire shape for a browsable OpenRouter model.
type openRouterModelDTO struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	ContextLength int    `json:"contextLength"`
}

// listOpenRouterModels returns the OpenRouter catalog sorted alphabetically by
// id (case-insensitive). The catalog is public (no OpenRouter account needed),
// but the route still sits behind this app's auth like every other endpoint.
func (s *Server) listOpenRouterModels(w http.ResponseWriter, r *http.Request) {
	models, err := s.openrouter.ListModels(r.Context())
	if err != nil {
		writeErr(w, err, http.StatusBadGateway)
		return
	}

	out := make([]openRouterModelDTO, len(models))
	for i, m := range models {
		out[i] = openRouterModelDTO{ID: m.ID, Name: m.Name, ContextLength: m.ContextLength}
	}
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i].ID) < strings.ToLower(out[j].ID)
	})

	writeJSON(w, http.StatusOK, out)
}
