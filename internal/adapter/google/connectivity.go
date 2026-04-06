package google

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

// connectivityCheckURL is a lightweight endpoint used by many clients for reachability probes.
const connectivityCheckURL = "https://connectivitycheck.gstatic.com/generate_204"

// Reachable reports whether a short outbound HTTPS request succeeds.
// On failure it wraps ErrOffline so callers can treat it like other network errors from this package.
func Reachable(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, connectivityCheckURL, nil)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrOffline, err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrOffline, err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode >= 400 {
		return fmt.Errorf("%w: unexpected HTTP status %s", ErrOffline, resp.Status)
	}
	return nil
}
