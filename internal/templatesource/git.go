package templatesource

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/config"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/storage/memory"
)

// GitSource fetches templates from a Git repository using standard git protocol.
// It works with any git remote (GitHub, GitLab, Bitbucket, self-hosted, etc.)
// and relies on the user's existing git credentials (SSH keys, credential helpers).
type GitSource struct {
	repoURL  string
	cloneDir string
}

// NewGitSource creates a template source from a git repository URL.
func NewGitSource(repoURL string) *GitSource {
	return &GitSource{repoURL: repoURL}
}

func (s *GitSource) ListVersions(_ context.Context, limit int) ([]string, error) {
	// Use ls-remote via go-git to list tags without cloning.
	rem := git.NewRemote(memory.NewStorage(), &config.RemoteConfig{
		Name: "origin",
		URLs: []string{s.repoURL},
	})

	refs, err := rem.List(&git.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list remote tags — make sure you can 'git clone %s': %w", s.repoURL, err)
	}

	var tags []string
	for _, ref := range refs {
		name := ref.Name()
		if name.IsTag() {
			tags = append(tags, name.Short())
		}
	}

	// Sort tags by semver descending (newest first).
	sort.Slice(tags, func(i, j int) bool {
		return compareSemver(tags[i], tags[j]) > 0
	})

	if limit > 0 && len(tags) > limit {
		tags = tags[:limit]
	}
	return tags, nil
}

func (s *GitSource) Download(_ context.Context, version string) (string, error) {
	tmpDir, err := os.MkdirTemp("", "spire-template-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}
	s.cloneDir = tmpDir

	cloneOpts := &git.CloneOptions{
		URL:   s.repoURL,
		Depth: 1,
	}

	// If a version is specified, clone that specific tag.
	if version != "" {
		cloneOpts.ReferenceName = plumbing.NewTagReferenceName(version)
		cloneOpts.SingleBranch = true
	}

	_, err = git.PlainClone(tmpDir, cloneOpts)
	if err != nil {
		_ = os.RemoveAll(tmpDir)
		return "", fmt.Errorf("failed to clone template — make sure you can 'git clone %s': %w", s.repoURL, err)
	}

	return tmpDir, nil
}

func (s *GitSource) Cleanup() {
	if s.cloneDir != "" {
		fmt.Println("🧹 Cleaning up temporary files...")
		_ = os.RemoveAll(s.cloneDir)
	}
}

// compareSemver compares two semver-like tag strings.
// Returns >0 if a > b, <0 if a < b, 0 if equal.
func compareSemver(a, b string) int {
	pa := parseSemverParts(a)
	pb := parseSemverParts(b)

	for i := 0; i < len(pa) && i < len(pb); i++ {
		if pa[i] != pb[i] {
			return pa[i] - pb[i]
		}
	}
	return len(pa) - len(pb)
}

// parseSemverParts extracts numeric parts from a version string.
// "v1.2.3" → [1, 2, 3], "1.0.0-beta" → [1, 0, 0]
func parseSemverParts(v string) []int {
	v = strings.TrimPrefix(v, "v")
	// Strip pre-release suffix.
	if idx := strings.IndexByte(v, '-'); idx >= 0 {
		v = v[:idx]
	}
	parts := strings.Split(v, ".")
	nums := make([]int, 0, len(parts))
	for _, p := range parts {
		n := 0
		for _, c := range p {
			if c >= '0' && c <= '9' {
				n = n*10 + int(c-'0')
			}
		}
		nums = append(nums, n)
	}
	return nums
}

var _ Source = (*GitSource)(nil)
