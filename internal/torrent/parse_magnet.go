package torrent

import (
	"errors"
	"fmt"
	"net/url"
)

func MagnetURIDisplayName(uri string) (string, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return "", fmt.Errorf("error parsing uri: %w", err)
	}
	if u.Scheme != "magnet" {
		return "", fmt.Errorf("unexpected scheme %q", u.Scheme)
	}
	q := u.Query()
	displayName := q.Get("dn")
	if displayName == "" {
		return "", errors.New("missing 'dn' query parameter")
	}
	return displayName, nil
}
