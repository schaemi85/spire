package templatesource

import "context"

// Source abstracts where scaffolding templates come from.
type Source interface {
	// ListVersions returns available template versions, ordered newest first.
	ListVersions(ctx context.Context, limit int) ([]string, error)
	// Download downloads the template for the given version and returns
	// the path to the extracted/ready directory.
	Download(ctx context.Context, version string) (string, error)
	// Cleanup removes any temporary files created during Download.
	Cleanup()
}
