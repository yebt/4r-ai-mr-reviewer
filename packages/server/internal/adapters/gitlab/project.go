package gitlab

import (
	"fmt"
	"net/url"
	"strings"
)

// ProjectPath extracts the namespaced project path ("group/project") from a
// repository URL, for use as the URL-encoded :id in the REST API.
func ProjectPath(repoURL string) (string, error) {
	u, err := url.Parse(repoURL)
	if err != nil {
		return "", fmt.Errorf("gitlab: parse repo url: %w", err)
	}
	p := strings.Trim(u.Path, "/")
	p = strings.TrimSuffix(p, ".git")
	if p == "" {
		return "", fmt.Errorf("gitlab: no project path in %q", repoURL)
	}
	return p, nil
}
