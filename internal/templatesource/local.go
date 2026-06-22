package templatesource

import (
	"context"
	"fmt"
	"os"
)

// LocalSource provides templates from a local filesystem directory.
type LocalSource struct {
	path string
}

// NewLocalSource creates a template source from a local directory path.
func NewLocalSource(path string) *LocalSource {
	return &LocalSource{path: path}
}

func (s *LocalSource) ListVersions(_ context.Context, _ int) ([]string, error) {
	return nil, fmt.Errorf("local template source does not support version listing")
}

func (s *LocalSource) Download(_ context.Context, _ string) (string, error) {
	info, err := os.Stat(s.path)
	if err != nil {
		return "", fmt.Errorf("template path does not exist: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("template path is not a directory: %s", s.path)
	}
	return s.path, nil
}

func (s *LocalSource) Cleanup() {
	// nothing to clean up for local sources
}

var _ Source = (*LocalSource)(nil)
