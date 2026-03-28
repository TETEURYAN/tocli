package google

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"golang.org/x/oauth2"
	googleoauth "golang.org/x/oauth2/google"
	calendar "google.golang.org/api/calendar/v3"
	tasks "google.golang.org/api/tasks/v1"
)

// clientID and clientSecret are injected at build time via -ldflags:
//
//	go build -ldflags "-X 'tocli/internal/adapter/google.clientID=<id>' \
//	                   -X 'tocli/internal/adapter/google.clientSecret=<secret>'"
//
// When both are empty the code falls back to reading credentials.json from
// the config directory (useful for development).
var (
	clientID     string
	clientSecret string
)

const oauthCallbackPort = 8085

var requiredScopes = []string{
	calendar.CalendarReadonlyScope,
	tasks.TasksScope,
}

type tokenDisk struct {
	AccessToken  string    `json:"access_token"`
	TokenType    string    `json:"token_type"`
	RefreshToken string    `json:"refresh_token"`
	Expiry       time.Time `json:"expiry"`
}

// buildOAuthConfig returns an OAuth2 config.
//   - If clientID/clientSecret are embedded (ldflags), uses them directly.
//   - Otherwise falls back to reading credentials.json from the config dir.
func buildOAuthConfig() (*oauth2.Config, error) {
	redirectURL := fmt.Sprintf("http://127.0.0.1:%d/callback", oauthCallbackPort)

	if clientID != "" && clientSecret != "" {
		return &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			Scopes:       requiredScopes,
			Endpoint:     googleoauth.Endpoint,
			RedirectURL:  redirectURL,
		}, nil
	}

	// Fallback: read credentials.json (developer workflow)
	path, err := CredentialsPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf(
			"Google credentials not configured.\n\n"+
				"  • If you are a developer: download credentials.json from Google Cloud Console\n"+
				"    and place it at %s\n\n"+
				"  • If you received this binary: please report it to the maintainer —\n"+
				"    the binary was not built with embedded credentials.",
			path,
		)
	}
	cfg, err := googleoauth.ConfigFromJSON(data, requiredScopes...)
	if err != nil {
		return nil, fmt.Errorf("parse credentials.json: %w", err)
	}
	cfg.RedirectURL = redirectURL
	return cfg, nil
}

func loadToken(path string) (*oauth2.Token, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var td tokenDisk
	if err := json.Unmarshal(data, &td); err != nil {
		return nil, err
	}
	return &oauth2.Token{
		AccessToken:  td.AccessToken,
		TokenType:    td.TokenType,
		RefreshToken: td.RefreshToken,
		Expiry:       td.Expiry,
	}, nil
}

func saveToken(path string, tok *oauth2.Token) error {
	td := tokenDisk{
		AccessToken:  tok.AccessToken,
		TokenType:    tok.TokenType,
		RefreshToken: tok.RefreshToken,
		Expiry:       tok.Expiry,
	}
	data, err := json.MarshalIndent(td, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

// Authenticate returns an HTTP client authorized for Google Calendar + Tasks.
//
// Flow:
//  1. Valid cached token found → returns immediately (silent).
//  2. Expired token with refresh token → auto-refreshes (silent).
//  3. No token → opens browser to Google consent page, waits for redirect.
func Authenticate(ctx context.Context) (*http.Client, error) {
	cfg, err := buildOAuthConfig()
	if err != nil {
		return nil, err
	}

	tokenPath, err := TokenPath()
	if err != nil {
		return nil, err
	}

	// Try cached token first.
	if tok, err := loadToken(tokenPath); err == nil && (tok.RefreshToken != "" || tok.AccessToken != "") {
		src := cfg.TokenSource(ctx, tok)
		newTok, err := src.Token()
		if err == nil {
			if newTok.RefreshToken == "" {
				newTok.RefreshToken = tok.RefreshToken
			}
			if newTok.AccessToken != tok.AccessToken || !newTok.Expiry.Equal(tok.Expiry) {
				_ = saveToken(tokenPath, newTok)
			}
			return oauth2.NewClient(ctx, src), nil
		}
		// Refresh failed (token revoked); fall through to full flow.
	}

	tok, err := runOAuthFlow(ctx, cfg)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(tokenPath), 0700); err != nil {
		return nil, err
	}
	if err := saveToken(tokenPath, tok); err != nil {
		return nil, fmt.Errorf("save token: %w", err)
	}
	return cfg.Client(ctx, tok), nil
}

func runOAuthFlow(ctx context.Context, cfg *oauth2.Config) (*oauth2.Token, error) {
	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			errCh <- fmt.Errorf("oauth callback: code not received")
			http.Error(w, "Missing code.", http.StatusBadRequest)
			return
		}
		codeCh <- code
		_, _ = w.Write([]byte(callbackHTML))
	})

	srv := &http.Server{
		Addr:              fmt.Sprintf("127.0.0.1:%d", oauthCallbackPort),
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()
	defer func() {
		sctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(sctx)
	}()

	authURL := cfg.AuthCodeURL("state", oauth2.AccessTypeOffline, oauth2.ApprovalForce)

	fmt.Fprintln(os.Stderr, authBanner)
	if err := openBrowser(authURL); err != nil {
		// Browser could not be opened automatically; show URL for manual copy.
		fmt.Fprintf(os.Stderr, "  Could not open browser automatically.\n  Copy and paste this URL:\n\n  %s\n\n", authURL)
	} else {
		fmt.Fprintf(os.Stderr, "  Browser opened. Complete the sign-in and return here.\n  (URL: %s)\n\n", authURL)
	}
	fmt.Fprintf(os.Stderr, "  Waiting for Google redirect on http://127.0.0.1:%d/callback ...\n\n", oauthCallbackPort)

	select {
	case code := <-codeCh:
		tok, err := cfg.Exchange(ctx, code)
		if err != nil {
			return nil, fmt.Errorf("exchange code: %w", err)
		}
		fmt.Fprintln(os.Stderr, "  ✓ Authentication successful! Starting tocli...\n")
		return tok, nil
	case err := <-errCh:
		return nil, err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// openBrowser tries to open url in the user's default browser.
func openBrowser(url string) error {
	var cmd string
	var args []string
	switch runtime.GOOS {
	case "linux":
		cmd = "xdg-open"
		args = []string{url}
	case "darwin":
		cmd = "open"
		args = []string{url}
	case "windows":
		cmd = "rundll32"
		args = []string{"url.dll,FileProtocolHandler", url}
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
	return exec.Command(cmd, args...).Start()
}

const authBanner = `
  ┌─────────────────────────────────────────────┐
  │   tocli — Google Authentication Required    │
  └─────────────────────────────────────────────┘

  tocli needs access to your Google account to show
  your Tasks and Calendar events.
`

const callbackHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>tocli – Authenticated</title>
  <style>
    body { font-family: sans-serif; display: flex; align-items: center;
           justify-content: center; height: 100vh; margin: 0;
           background: #0f0f10; color: #e0e0e0; }
    .card { text-align: center; padding: 2rem 3rem; border-radius: 12px;
            border: 1px solid #2a2a35; background: #1a1a22; }
    h1 { color: #a78bfa; margin-bottom: .5rem; }
    p  { color: #888; }
  </style>
</head>
<body>
  <div class="card">
    <h1>✓ Authentication successful</h1>
    <p>You can close this tab and return to the terminal.</p>
  </div>
</body>
</html>`
