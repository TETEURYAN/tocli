package google

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

// CredentialsPath returns the path to credentials.json.
// Used only as a developer fallback when the binary was not built with
// embedded client credentials (ldflags). Most end-users never need this file.
// Override with env var TOC_GOOGLE_CREDENTIALS.
func CredentialsPath() (string, error) {
	if p := os.Getenv("TOC_GOOGLE_CREDENTIALS"); p != "" {
		return p, nil
	}
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "credentials.json"), nil
}

// TokenPath returns where the OAuth token is persisted between runs.
func TokenPath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "token.json"), nil
}
