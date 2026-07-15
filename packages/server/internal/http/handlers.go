package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/webcloster-dev/ai-reviewer/internal/adapters/gitlab"
	apphumanize "github.com/webcloster-dev/ai-reviewer/internal/app/humanize"
	"github.com/webcloster-dev/ai-reviewer/internal/app/profiles"
	"github.com/webcloster-dev/ai-reviewer/internal/app/providers"
	"github.com/webcloster-dev/ai-reviewer/internal/app/repos"
	"github.com/webcloster-dev/ai-reviewer/internal/app/reviews"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/account"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/profile"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/provider"
	domainrepo "github.com/webcloster-dev/ai-reviewer/internal/domain/repo"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/review"
)

// --- skills ---

type skillsResp struct {
	Risk        string `json:"risk"`
	Readability string `json:"readability"`
	Reliability string `json:"reliability"`
	Resilience  string `json:"resilience"`
}

func (s *Server) getSkills(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, skillsResp{
		Risk:        s.skills.Risk,
		Readability: s.skills.Readability,
		Reliability: s.skills.Reliability,
		Resilience:  s.skills.Resilience,
	})
}

// --- accounts ---

func (s *Server) createAccount(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name    string `json:"name"`
		BaseURL string `json:"baseUrl"`
		Token   string `json:"token"`
	}
	if err := decode(r, &in); err != nil {
		writeErr(w, err, http.StatusBadRequest)
		return
	}
	a, err := s.accounts.Add(r.Context(), in.Name, in.BaseURL, in.Token)
	if err != nil {
		writeErr(w, err, http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusCreated, toAccount(a))
}

func (s *Server) listAccounts(w http.ResponseWriter, r *http.Request) {
	as, err := s.accounts.List(r.Context())
	if err != nil {
		writeErr(w, err, http.StatusInternalServerError)
		return
	}
	out := make([]accountResp, 0, len(as))
	for _, a := range as {
		out = append(out, toAccount(a))
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) deleteAccount(w http.ResponseWriter, r *http.Request) {
	if err := s.accounts.Remove(r.Context(), r.PathValue("id")); err != nil {
		writeErr(w, err, http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- providers ---

func (s *Server) createProvider(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name        string   `json:"name"`
		Kind        string   `json:"kind"`
		BaseURL     string   `json:"baseUrl"`
		Model       string   `json:"model"`
		APIKey      string   `json:"apiKey"`
		MakeDefault bool     `json:"makeDefault"`
		Temperature *float64 `json:"temperature"`
		Models      []string `json:"models"`
	}
	if err := decode(r, &in); err != nil {
		writeErr(w, err, http.StatusBadRequest)
		return
	}
	p, err := s.providers.Add(r.Context(), providers.AddInput{
		Name: in.Name, Kind: provider.Kind(in.Kind), BaseURL: in.BaseURL,
		Model: in.Model, APIKey: in.APIKey, MakeDefault: in.MakeDefault,
		Temperature: in.Temperature, Models: in.Models,
	})
	if err != nil {
		writeErr(w, err, http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusCreated, toProvider(p))
}

func (s *Server) listProviders(w http.ResponseWriter, r *http.Request) {
	ps, err := s.providers.List(r.Context())
	if err != nil {
		writeErr(w, err, http.StatusInternalServerError)
		return
	}
	out := make([]providerResp, 0, len(ps))
	for _, p := range ps {
		out = append(out, toProvider(p))
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) updateProvider(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name        string   `json:"name"`
		Kind        string   `json:"kind"`
		BaseURL     string   `json:"baseUrl"`
		Model       string   `json:"model"`
		APIKey      string   `json:"apiKey"`
		Temperature *float64 `json:"temperature"`
		Models      []string `json:"models"`
	}
	if err := decode(r, &in); err != nil {
		writeErr(w, err, http.StatusBadRequest)
		return
	}
	p, err := s.providers.Update(r.Context(), r.PathValue("id"), providers.UpdateInput{
		Name: in.Name, Kind: provider.Kind(in.Kind), BaseURL: in.BaseURL, Model: in.Model, APIKey: in.APIKey,
		Temperature: in.Temperature, Models: in.Models,
	})
	if err != nil {
		writeErr(w, err, http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, toProvider(p))
}

func (s *Server) setDefaultProvider(w http.ResponseWriter, r *http.Request) {
	if err := s.providers.SetDefault(r.Context(), r.PathValue("id")); err != nil {
		writeErr(w, err, http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) deleteProvider(w http.ResponseWriter, r *http.Request) {
	if err := s.providers.Remove(r.Context(), r.PathValue("id")); err != nil {
		writeErr(w, err, http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- profiles ---

func (s *Server) createProfile(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name      string   `json:"name"`
		Language  string   `json:"language"`
		Formality string   `json:"formality"`
		Emojis    bool     `json:"emojis"`
		Samples   []string `json:"samples"`
	}
	if err := decode(r, &in); err != nil {
		writeErr(w, err, http.StatusBadRequest)
		return
	}
	p, err := s.profiles.Add(r.Context(), profiles.AddInput{
		Name: in.Name, Language: in.Language, Formality: in.Formality,
		Emojis: in.Emojis, Samples: in.Samples,
	})
	if err != nil {
		writeErr(w, err, http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusCreated, toProfile(p))
}

func (s *Server) listProfiles(w http.ResponseWriter, r *http.Request) {
	ps, err := s.profiles.List(r.Context())
	if err != nil {
		writeErr(w, err, http.StatusInternalServerError)
		return
	}
	out := make([]profileResp, 0, len(ps))
	for _, p := range ps {
		out = append(out, toProfile(p))
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) getProfile(w http.ResponseWriter, r *http.Request) {
	p, err := s.profiles.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		writeErr(w, err, http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, toProfile(p))
}

func (s *Server) updateProfile(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name      string   `json:"name"`
		Language  string   `json:"language"`
		Formality string   `json:"formality"`
		Emojis    bool     `json:"emojis"`
		Samples   []string `json:"samples"`
	}
	if err := decode(r, &in); err != nil {
		writeErr(w, err, http.StatusBadRequest)
		return
	}
	p, err := s.profiles.Update(r.Context(), r.PathValue("id"), profiles.UpdateInput{
		Name: in.Name, Language: in.Language, Formality: in.Formality,
		Emojis: in.Emojis, Samples: in.Samples,
	})
	if err != nil {
		writeErr(w, err, http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, toProfile(p))
}

func (s *Server) deleteProfile(w http.ResponseWriter, r *http.Request) {
	if err := s.profiles.Delete(r.Context(), r.PathValue("id")); err != nil {
		writeErr(w, err, http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) redistillProfile(w http.ResponseWriter, r *http.Request) {
	if err := s.profiles.Redistill(r.Context(), r.PathValue("id")); err != nil {
		writeErr(w, err, http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": profile.StyleStatusPending})
}

// --- repos ---

func (s *Server) createRepo(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name       string `json:"name"`
		URL        string `json:"url"`
		AccountID  string `json:"accountId"`
		ProviderID string `json:"providerId"`
		Model      string `json:"model"`
	}
	if err := decode(r, &in); err != nil {
		writeErr(w, err, http.StatusBadRequest)
		return
	}
	rp, err := s.repos.Add(r.Context(), repos.AddInput{
		Name: in.Name, URL: in.URL, AccountID: in.AccountID, ProviderID: in.ProviderID, Model: in.Model,
	})
	if err != nil {
		writeErr(w, err, http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusCreated, toRepo(rp))
}

func (s *Server) listRepos(w http.ResponseWriter, r *http.Request) {
	rs, err := s.repos.List(r.Context())
	if err != nil {
		writeErr(w, err, http.StatusInternalServerError)
		return
	}
	out := make([]repoResp, 0, len(rs))
	for _, rp := range rs {
		out = append(out, toRepo(rp))
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) assignRepo(w http.ResponseWriter, r *http.Request) {
	var in struct {
		ProviderID string `json:"providerId"`
		Model      string `json:"model"`
	}
	if err := decode(r, &in); err != nil {
		writeErr(w, err, http.StatusBadRequest)
		return
	}
	rp, err := s.repos.Assign(r.Context(), r.PathValue("id"), in.ProviderID, in.Model)
	if err != nil {
		writeErr(w, err, http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, toRepo(rp))
}

func (s *Server) deleteRepo(w http.ResponseWriter, r *http.Request) {
	if err := s.repos.Remove(r.Context(), r.PathValue("id")); err != nil {
		writeErr(w, err, http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) listMergeRequests(w http.ResponseWriter, r *http.Request) {
	mrs, err := s.reviews.ListOpenMergeRequests(r.Context(), r.PathValue("id"))
	if err != nil {
		writeErr(w, err, http.StatusBadGateway)
		return
	}
	out := make([]mrResp, 0, len(mrs))
	for _, m := range mrs {
		out = append(out, toMR(m))
	}
	writeJSON(w, http.StatusOK, out)
}

// --- reviews ---

func (s *Server) listReviews(w http.ResponseWriter, r *http.Request) {
	list := s.reviews.List
	if q := r.URL.Query().Get("archived"); q == "1" || q == "true" {
		list = s.reviews.ListArchived
	}
	rvs, err := list(r.Context(), r.PathValue("id"))
	if err != nil {
		writeErr(w, err, http.StatusInternalServerError)
		return
	}
	out := make([]reviewResp, 0, len(rvs))
	for _, rv := range rvs {
		out = append(out, toReview(rv))
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) createReview(w http.ResponseWriter, r *http.Request) {
	var in struct {
		RepoID string `json:"repoId"`
		MRIID  int    `json:"mrIid"`
		Mode   string `json:"mode"`
	}
	if err := decode(r, &in); err != nil {
		writeErr(w, err, http.StatusBadRequest)
		return
	}
	rv, err := s.reviews.Create(r.Context(), in.RepoID, in.MRIID, review.ContextMode(in.Mode))
	if err != nil {
		writeErr(w, err, http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusCreated, toReview(rv))
}

func (s *Server) getReview(w http.ResponseWriter, r *http.Request) {
	rv, err := s.reviews.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		writeErr(w, err, http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, toReview(rv))
}

func (s *Server) retryReview(w http.ResponseWriter, r *http.Request) {
	rv, err := s.reviews.Retry(r.Context(), r.PathValue("id"))
	if err != nil {
		writeErr(w, err, http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusCreated, toReview(rv))
}

func (s *Server) deleteReview(w http.ResponseWriter, r *http.Request) {
	if err := s.reviews.Delete(r.Context(), r.PathValue("id")); err != nil {
		writeErr(w, err, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) cancelReview(w http.ResponseWriter, r *http.Request) {
	if err := s.reviews.Cancel(r.Context(), r.PathValue("id")); err != nil {
		if errors.Is(err, reviews.ErrNotCancelable) {
			writeErr(w, err, http.StatusConflict)
			return
		}
		writeErr(w, err, http.StatusInternalServerError)
		return
	}
	// Cancellation is cooperative: a running review flips to "cancelled"
	// shortly after, which the client observes by polling.
	writeJSON(w, http.StatusOK, map[string]string{"status": "cancelling"})
}

func (s *Server) archiveReview(w http.ResponseWriter, r *http.Request) {
	if err := s.reviews.Archive(r.Context(), r.PathValue("id")); err != nil {
		writeErr(w, err, http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "archived"})
}

func (s *Server) unarchiveReview(w http.ResponseWriter, r *http.Request) {
	if err := s.reviews.Unarchive(r.Context(), r.PathValue("id")); err != nil {
		writeErr(w, err, http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "unarchived"})
}

func (s *Server) publishReview(w http.ResponseWriter, r *http.Request) {
	var in struct {
		All              bool    `json:"all"`
		Indices          []int   `json:"indices"`
		IncludeSummary   *bool   `json:"includeSummary"`
		SummaryOverride  *string `json:"summaryOverride"`
		FindingOverrides []struct {
			Index int    `json:"index"`
			Text  string `json:"text"`
		} `json:"findingOverrides"`
	}
	if err := decode(r, &in); err != nil {
		writeErr(w, err, http.StatusBadRequest)
		return
	}
	var findingOverrides map[int]string
	if len(in.FindingOverrides) > 0 {
		findingOverrides = make(map[int]string, len(in.FindingOverrides))
		for _, o := range in.FindingOverrides {
			findingOverrides[o.Index] = o.Text
		}
	}
	if err := s.reviews.Publish(r.Context(), r.PathValue("id"), reviews.Selection{
		All:              in.All,
		Indices:          in.Indices,
		IncludeSummary:   in.IncludeSummary,
		SummaryOverride:  in.SummaryOverride,
		FindingOverrides: findingOverrides,
	}); err != nil {
		writeErr(w, err, http.StatusBadGateway)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "published"})
}

// humanizeReview rewrites ONE target of a finished review — a single finding's
// issue/why/fix parts, or the summary — in a profile's author voice, returning
// the structured parts. It is ephemeral: nothing is persisted. The frontend
// fires one call per target so each rewrite is independent.
//
// Request: {profileId, target:"finding", index} or {profileId, target:"summary"}.
// Error mapping: unknown review/profile → 404; review not done or style guide not
// ready → 409; unknown/missing target or out-of-range index → 400; LLM/parse
// failure → 502.
func (s *Server) humanizeReview(w http.ResponseWriter, r *http.Request) {
	var in struct {
		ProfileID string `json:"profileId"`
		Target    string `json:"target"`
		Index     int    `json:"index"`
	}
	if err := decode(r, &in); err != nil {
		writeErr(w, err, http.StatusBadRequest)
		return
	}

	switch in.Target {
	case "summary":
		out, err := s.humanize.HumanizeSummary(r.Context(), r.PathValue("id"), in.ProfileID)
		if err != nil {
			s.writeHumanizeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	case "finding":
		out, err := s.humanize.HumanizeFinding(r.Context(), r.PathValue("id"), in.ProfileID, in.Index)
		if err != nil {
			s.writeHumanizeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	default:
		writeErr(w, errors.New(`humanize: target must be "finding" or "summary"`), http.StatusBadRequest)
	}
}

// getHumanizations returns every persisted humanize run of a review, grouped for
// the SPA to rehydrate its tabs: the summary rewrites as an ordered list, and the
// finding rewrites keyed by finding index (as a string, since JSON object keys
// are strings). Within each group the tabs preserve their run order.
func (s *Server) getHumanizations(w http.ResponseWriter, r *http.Request) {
	hs, err := s.humanize.List(r.Context(), r.PathValue("id"))
	if err != nil {
		writeErr(w, err, http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, toHumanizations(hs))
}

// writeHumanizeErr maps humanize service errors to HTTP status codes: unknown
// review/profile → 404 (via writeErr); review not done / style guide not ready →
// 409; out-of-range finding index → 400; any other failure (LLM/parse) → 502.
func (s *Server) writeHumanizeErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, apphumanize.ErrReviewNotDone), errors.Is(err, apphumanize.ErrStyleGuideNotReady):
		writeErr(w, err, http.StatusConflict)
	case errors.Is(err, apphumanize.ErrFindingIndexOutOfRange):
		writeErr(w, err, http.StatusBadRequest)
	default:
		// isNotFound (via writeErr) still maps unknown review/profile to 404;
		// any other failure here is an upstream LLM/parse error.
		writeErr(w, err, http.StatusBadGateway)
	}
}

// --- response DTOs ---

type accountResp struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	BaseURL   string    `json:"baseUrl"`
	CreatedAt time.Time `json:"createdAt"`
}

func toAccount(a account.Account) accountResp {
	return accountResp{ID: a.ID, Name: a.Name, BaseURL: a.BaseURL, CreatedAt: a.CreatedAt}
}

type providerResp struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Kind        string    `json:"kind"`
	BaseURL     string    `json:"baseUrl"`
	Model       string    `json:"model"`
	IsDefault   bool      `json:"isDefault"`
	Temperature *float64  `json:"temperature"`
	Models      []string  `json:"models"`
	CreatedAt   time.Time `json:"createdAt"`
}

func toProvider(p provider.Provider) providerResp {
	models := p.Models
	if models == nil {
		models = []string{}
	}
	return providerResp{
		ID: p.ID, Name: p.Name, Kind: string(p.Kind), BaseURL: p.BaseURL,
		Model: p.Model, IsDefault: p.IsDefault, Temperature: p.Temperature,
		Models: models, CreatedAt: p.CreatedAt,
	}
}

type profileResp struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	Language         string    `json:"language"`
	Formality        string    `json:"formality"`
	Emojis           bool      `json:"emojis"`
	Samples          []string  `json:"samples"`
	StyleGuide       string    `json:"styleGuide"`
	StyleGuideStatus string    `json:"styleGuideStatus"`
	StyleGuideError  string    `json:"styleGuideError"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

func toProfile(p profile.Profile) profileResp {
	samples := p.Samples
	if samples == nil {
		samples = []string{}
	}
	return profileResp{
		ID: p.ID, Name: p.Name, Language: p.Language, Formality: p.Formality,
		Emojis: p.Emojis, Samples: samples, StyleGuide: p.StyleGuide,
		StyleGuideStatus: p.StyleGuideStatus, StyleGuideError: p.StyleGuideError,
		CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt,
	}
}

type repoResp struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	URL        string    `json:"url"`
	AccountID  string    `json:"accountId"`
	ProviderID string    `json:"providerId"`
	Model      string    `json:"model"`
	CreatedAt  time.Time `json:"createdAt"`
}

func toRepo(rp domainrepo.Repo) repoResp {
	return repoResp{
		ID: rp.ID, Name: rp.Name, URL: rp.URL, AccountID: rp.AccountID,
		ProviderID: rp.ProviderID, Model: rp.Model, CreatedAt: rp.CreatedAt,
	}
}

type mrResp struct {
	IID          int    `json:"iid"`
	Title        string `json:"title"`
	State        string `json:"state"`
	SourceBranch string `json:"sourceBranch"`
	TargetBranch string `json:"targetBranch"`
	WebURL       string `json:"webUrl"`
	Author       string `json:"author"`
}

func toMR(m gitlab.MergeRequest) mrResp {
	return mrResp{
		IID: m.IID, Title: m.Title, State: m.State, SourceBranch: m.SourceBranch,
		TargetBranch: m.TargetBranch, WebURL: m.WebURL, Author: m.Author.Username,
	}
}

type findingResp struct {
	Index     int    `json:"index"`
	Dimension string `json:"dimension"`
	Severity  string `json:"severity"`
	File      string `json:"file"`
	Line      int    `json:"line"`
	Issue     string `json:"issue"`
	Why       string `json:"why"`
	Fix       string `json:"fix"`
	Blocking  bool   `json:"blocking"`
	Published bool   `json:"published"`
}

type reviewResp struct {
	ID               string        `json:"id"`
	RepoID           string        `json:"repoId"`
	MRIID            int           `json:"mrIid"`
	ContextMode      string        `json:"contextMode"`
	Status           string        `json:"status"`
	Phase            string        `json:"phase"`
	Archived         bool          `json:"archived"`
	SummaryPublished bool          `json:"summaryPublished"`
	Summary          string        `json:"summary"`
	Recommendation   string        `json:"recommendation"`
	Score            int           `json:"score"`
	Error            string        `json:"error,omitempty"`
	InputTokens      int           `json:"inputTokens"`
	OutputTokens     int           `json:"outputTokens"`
	Findings         []findingResp `json:"findings"`
	CreatedAt        time.Time     `json:"createdAt"`
	UpdatedAt        time.Time     `json:"updatedAt"`
}

func toReview(rv review.Review) reviewResp {
	findings := make([]findingResp, 0, len(rv.Findings))
	for i, f := range rv.Findings {
		findings = append(findings, findingResp{
			Index: i, Dimension: string(f.Dimension), Severity: string(f.Severity),
			File: f.File, Line: f.Line, Issue: f.Issue, Why: f.Why, Fix: f.Fix,
			Blocking: f.Blocking, Published: f.Published,
		})
	}
	return reviewResp{
		ID: rv.ID, RepoID: rv.RepoID, MRIID: rv.MRIID, ContextMode: string(rv.ContextMode),
		Status: string(rv.Status), Phase: rv.Phase, Archived: rv.Archived, SummaryPublished: rv.SummaryPublished, Summary: rv.Summary, Recommendation: string(rv.Recommendation),
		Score: rv.Score, Error: rv.Error, InputTokens: rv.InputTokens, OutputTokens: rv.OutputTokens,
		Findings: findings, CreatedAt: rv.CreatedAt, UpdatedAt: rv.UpdatedAt,
	}
}

// summaryHumanizedResp is one persisted summary rewrite tab.
type summaryHumanizedResp struct {
	Summary string `json:"summary"`
}

// findingHumanizedResp is one persisted finding rewrite tab.
type findingHumanizedResp struct {
	Issue string `json:"issue"`
	Why   string `json:"why"`
	Fix   string `json:"fix"`
}

// humanizationsResp is the grouped, rehydration-ready view of a review's
// persisted humanizations. Findings are keyed by finding index (stringified for
// JSON object keys); tabs within each group preserve their run order.
type humanizationsResp struct {
	Summary  []summaryHumanizedResp            `json:"summary"`
	Findings map[string][]findingHumanizedResp `json:"findings"`
}

func toHumanizations(hs []review.Humanization) humanizationsResp {
	out := humanizationsResp{
		Summary:  []summaryHumanizedResp{},
		Findings: map[string][]findingHumanizedResp{},
	}
	for _, h := range hs {
		switch h.Target {
		case review.HumanizationSummary:
			out.Summary = append(out.Summary, summaryHumanizedResp{Summary: h.Summary})
		case review.HumanizationFinding:
			key := strconv.Itoa(h.FindingIndex)
			out.Findings[key] = append(out.Findings[key], findingHumanizedResp{Issue: h.Issue, Why: h.Why, Fix: h.Fix})
		}
	}
	return out
}
