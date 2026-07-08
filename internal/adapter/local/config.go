// Package local implements domain repositories backed by files under the
// user's config directory, for data that has no Google equivalent (e.g.
// daily ratings) and therefore isn't handled by internal/adapter/google.
package local

import (
	"os"
	"path/filepath"
)

const appDirName = "tocli"

// ConfigDir returns ~/.config/tocli (or $XDG_CONFIG_HOME/tocli).
func ConfigDir() (string, error) {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, appDirName), nil
}

// RatingsPath returns where daily ratings are persisted between runs.
// Override with the TOC_RATINGS_PATH environment variable.
func RatingsPath() (string, error) {
	if p := os.Getenv("TOC_RATINGS_PATH"); p != "" {
		return p, nil
	}
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "ratings.json"), nil
}
