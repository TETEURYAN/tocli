package google

import (
	"errors"
	"fmt"
	"net"

	"google.golang.org/api/googleapi"
)

var (
	ErrNotAuthenticated = errors.New("google: not authenticated (check credentials and token)")
	ErrOffline          = errors.New("google: network unreachable")
	ErrRateLimited      = errors.New("google: rate limited; try again later")
	ErrNotFound         = errors.New("google: resource not found")
	ErrForbidden        = errors.New("google: access denied (scopes or API not enabled)")
)

func wrapAPIError(operation string, err error) error {
	if err == nil {
		return nil
	}

	var netErr *net.OpError
	if errors.As(err, &netErr) {
		return fmt.Errorf("%s: %w: %v", operation, ErrOffline, err)
	}

	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return fmt.Errorf("%s: %w: %v", operation, ErrOffline, err)
	}

	var apiErr *googleapi.Error
	if errors.As(err, &apiErr) {
		switch apiErr.Code {
		case 401:
			return fmt.Errorf("%s: %w", operation, ErrNotAuthenticated)
		case 403:
			return fmt.Errorf("%s: %w", operation, ErrForbidden)
		case 404:
			return fmt.Errorf("%s: %w", operation, ErrNotFound)
		case 429:
			return fmt.Errorf("%s: %w", operation, ErrRateLimited)
		default:
			return fmt.Errorf("%s: google api error %d: %s", operation, apiErr.Code, apiErr.Message)
		}
	}

	return fmt.Errorf("%s: %w", operation, err)
}
