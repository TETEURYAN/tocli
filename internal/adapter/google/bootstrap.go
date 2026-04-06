package google

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"tocli/internal/domain"
)

// Repos holds concrete Google-backed repositories.
type Repos struct {
	Tasks    *TaskRepo
	Calendar *CalendarRepo
}

// NewRepos builds Task and Calendar repositories with an authorized HTTP client.
func NewRepos(ctx context.Context, httpClient *http.Client) (*Repos, error) {
	client, err := NewClient(ctx, httpClient)
	if err != nil {
		return nil, err
	}
	return &Repos{
		Tasks:    NewTaskRepo(ctx, client),
		Calendar: NewCalendarRepo(ctx, client),
	}, nil
}

// TryGoogleRepos runs the full auth flow and returns ready-to-use repositories.
// Returns a non-nil error if authentication or client setup fails.
func TryGoogleRepos(ctx context.Context) (*Repos, error) {
	httpClient, err := Authenticate(ctx)
	if err != nil {
		return nil, err
	}
	return NewRepos(ctx, httpClient)
}

// ReposAsInterfaces returns the Google adapters as domain interfaces.
func ReposAsInterfaces(r *Repos) (domain.TaskRepository, domain.EventRepository) {
	if r == nil {
		return nil, nil
	}
	return r.Tasks, r.Calendar
}

// WarnFallback writes a short diagnostic to stderr and explains the fallback.
func WarnFallback(reason error) {
	headline := "  tocli: running in offline mode (mock data)."
	if errors.Is(reason, ErrOffline) {
		headline = "  tocli: no internet connection; running in offline mode (mock data)."
	}
	fmt.Fprintf(os.Stderr,
		"\n%s\n"+
			"  Reason: %v\n"+
			"  Run with -offline to suppress this message.\n"+
			"  See docs/GOOGLE.md to set up Google integration.\n\n",
		headline,
		reason,
	)
}
