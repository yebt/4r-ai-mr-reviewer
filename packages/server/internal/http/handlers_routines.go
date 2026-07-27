package httpapi

import (
	"net/http"

	"github.com/webcloster-dev/ai-reviewer/internal/domain/routine"
)

// repoPreflight reports which merge-request routine actions the repo's token and
// access level permit. Unknown repo → 404; upstream GitLab failure → 502.
func (s *Server) repoPreflight(w http.ResponseWriter, r *http.Request) {
	pf, err := s.routines.Preflight(r.Context(), r.PathValue("id"))
	if err != nil {
		writeErr(w, err, http.StatusBadGateway)
		return
	}
	writeJSON(w, http.StatusOK, toPreflight(pf))
}

// --- response DTOs ---

type preflightCheckResp struct {
	Capability string `json:"capability"`
	Label      string `json:"label"`
	Status     string `json:"status"`
	Detail     string `json:"detail"`
}

type preflightResp struct {
	TokenScopes     []string             `json:"tokenScopes"`
	ScopesKnown     bool                 `json:"scopesKnown"`
	AccessLevel     int                  `json:"accessLevel"`
	AccessLevelName string               `json:"accessLevelName"`
	DefaultBranch   string               `json:"defaultBranch"`
	Checks          []preflightCheckResp `json:"checks"`
}

func toPreflight(pf routine.Preflight) preflightResp {
	scopes := pf.TokenScopes
	if scopes == nil {
		scopes = []string{}
	}
	checks := make([]preflightCheckResp, 0, len(pf.Checks))
	for _, c := range pf.Checks {
		checks = append(checks, preflightCheckResp{
			Capability: string(c.Capability), Label: c.Label, Status: string(c.Status), Detail: c.Detail,
		})
	}
	return preflightResp{
		TokenScopes:     scopes,
		ScopesKnown:     pf.ScopesKnown,
		AccessLevel:     pf.AccessLevel,
		AccessLevelName: pf.AccessLevelName,
		DefaultBranch:   pf.DefaultBranch,
		Checks:          checks,
	}
}
