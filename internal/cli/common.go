package cli

import (
	"path/filepath"

	"agent-os/internal/project"
)

func resolveProjectRoot(path string) (string, error) {
	if path != "" && path != "." {
		return filepath.Abs(path)
	}

	return project.DiscoverRoot(".")
}
