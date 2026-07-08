package httpapi

import (
	"net/http"
	"time"

	"github.com/webcloster-dev/ai-reviewer/internal/adapters/gitlab"
	"github.com/webcloster-dev/ai-reviewer/internal/app/providers"
	"github.com/webcloster-dev/ai-reviewer/internal/app/repos"
	"github.com/webcloster-dev/ai-reviewer/internal/app/reviews"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/account"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/provider"
	domainrepo "github.com/webcloster-dev/ai-reviewer/internal/domain/repo"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/review"
)

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
		Name        string `json:"name"`
		Kind        string `json:"kind"`
		BaseURL     string `json:"baseUrl"`
		Model       string `json:"model"`
		APIKey      string `json:"apiKey"`
		MakeDefault bool   `json:"makeDefault"`
	}
	if err := decode(r, &in); err != nil {
		writeErr(w, err, http.StatusBadRequest)
		return
	}
	p, err := s.providers.Add(r.Context(), providers.AddInput{
		Name: in.Name, Kind: provider.Kind(in.Kind), BaseURL: in.BaseURL,
		Model: in.Model, APIKey: in.APIKey, MakeDefault: in.MakeDefault,
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
	rvs, err := s.reviews.List(r.Context(), r.PathValue("id"))
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

func (s *Server) publishReview(w http.ResponseWriter, r *http.Request) {
	var in struct {
		All     bool  `json:"all"`
		Indices []int `json:"indices"`
	}
	if err := decode(r, &in); err != nil {
		writeErr(w, err, http.StatusBadRequest)
		return
	}
	if err := s.reviews.Publish(r.Context(), r.PathValue("id"), reviews.Selection{All: in.All, Indices: in.Indices}); err != nil {
		writeErr(w, err, http.StatusBadGateway)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "published"})
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
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Kind      string    `json:"kind"`
	BaseURL   string    `json:"baseUrl"`
	Model     string    `json:"model"`
	IsDefault bool      `json:"isDefault"`
	CreatedAt time.Time `json:"createdAt"`
}

func toProvider(p provider.Provider) providerResp {
	return providerResp{
		ID: p.ID, Name: p.Name, Kind: string(p.Kind), BaseURL: p.BaseURL,
		Model: p.Model, IsDefault: p.IsDefault, CreatedAt: p.CreatedAt,
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
	ID             string        `json:"id"`
	RepoID         string        `json:"repoId"`
	MRIID          int           `json:"mrIid"`
	ContextMode    string        `json:"contextMode"`
	Status         string        `json:"status"`
	Summary        string        `json:"summary"`
	Recommendation string        `json:"recommendation"`
	Score          int           `json:"score"`
	Error          string        `json:"error,omitempty"`
	InputTokens    int           `json:"inputTokens"`
	OutputTokens   int           `json:"outputTokens"`
	Findings       []findingResp `json:"findings"`
	CreatedAt      time.Time     `json:"createdAt"`
	UpdatedAt      time.Time     `json:"updatedAt"`
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
		Status: string(rv.Status), Summary: rv.Summary, Recommendation: string(rv.Recommendation),
		Score: rv.Score, Error: rv.Error, InputTokens: rv.InputTokens, OutputTokens: rv.OutputTokens,
		Findings: findings, CreatedAt: rv.CreatedAt, UpdatedAt: rv.UpdatedAt,
	}
}
