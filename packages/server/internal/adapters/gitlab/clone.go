package gitlab

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Cloner performs shallow clones of GitLab repositories using the system git.
//
// The access token is never placed in the clone URL or command arguments
// (which would leak it to `ps` and process listings). Instead it is provided
// through GIT_ASKPASS: git invokes a tiny helper script that echoes the token
// from an environment variable only this child process can see.
type Cloner struct {
	token string
}

// NewCloner builds a Cloner. An empty token clones public/anonymous repos.
func NewCloner(token string) *Cloner {
	return &Cloner{token: token}
}

// Clone shallow-clones repoURL at ref into a "repo" subdirectory of workDir and
// returns the checkout path. workDir must exist and is owned by the caller
// (typically a temp dir it later removes).
func (c *Cloner) Clone(ctx context.Context, repoURL, ref, workDir string) (string, error) {
	if repoURL == "" {
		return "", fmt.Errorf("gitlab clone: empty repo URL")
	}
	dest := filepath.Join(workDir, "repo")

	args := []string{"clone", "--depth", "1"}
	if ref != "" {
		if err := validateRef(ref); err != nil {
			return "", err
		}
		args = append(args, "--branch", ref)
	}

	cloneURL := repoURL
	env := baseGitEnv()

	if c.token != "" {
		withUser, err := injectUsername(repoURL, "oauth2")
		if err != nil {
			return "", err
		}
		cloneURL = withUser

		askpass, err := writeAskpass(workDir)
		if err != nil {
			return "", err
		}
		env = append(env, "GIT_ASKPASS="+askpass, "GIT_TOKEN="+c.token)
	}

	args = append(args, cloneURL, dest)

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Env = env
	var stderr strings.Builder
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("gitlab clone: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return dest, nil
}

// validateRef guards the ref passed to `git clone --branch`. It comes from the
// merge request's source branch (influenceable by whoever opens the MR), so
// reject anything that could be read as an option (leading '-') or that falls
// outside a conservative git-ref charset, rather than relying on git/GitLab's own
// ref-format enforcement.
func validateRef(ref string) error {
	if strings.HasPrefix(ref, "-") || strings.Contains(ref, "..") {
		return fmt.Errorf("gitlab clone: invalid ref %q", ref)
	}
	for _, r := range ref {
		ok := r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' ||
			r == '.' || r == '_' || r == '/' || r == '-'
		if !ok {
			return fmt.Errorf("gitlab clone: invalid ref %q", ref)
		}
	}
	return nil
}

// baseGitEnv returns the process environment with interactive prompts disabled,
// so a missing credential fails fast instead of hanging.
func baseGitEnv() []string {
	return append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
}

// injectUsername adds a username to an http(s) URL so git asks GIT_ASKPASS for
// the matching password (the token). Non-http schemes (file://, ssh) are left
// untouched.
func injectUsername(rawURL, user string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("gitlab clone: parse url: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return rawURL, nil
	}
	u.User = url.User(user)
	return u.String(), nil
}

// writeAskpass writes a 0700 helper script that prints $GIT_TOKEN, keeping the
// token out of command arguments.
func writeAskpass(dir string) (string, error) {
	path := filepath.Join(dir, "askpass.sh")
	const script = "#!/bin/sh\nprintf '%s' \"$GIT_TOKEN\"\n"
	if err := os.WriteFile(path, []byte(script), 0o700); err != nil {
		return "", fmt.Errorf("gitlab clone: write askpass: %w", err)
	}
	return path, nil
}
