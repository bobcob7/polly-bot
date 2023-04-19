package torrent

import (
	"errors"
	"fmt"
	"net/url"
)

var errMissingDNQuery = errors.New("missing 'dn' query parameter")

type unexpectedSchemeError struct {
	scheme string
}

func (u unexpectedSchemeError) Error() string {
	return fmt.Sprintf("unexpected scheme got=%q want=%q", u.scheme, "magnet")
}

func MagnetURIDisplayName(uri string) (string, error) {
	parsedURI, err := url.Parse(uri)
	if err != nil {
		return "", fmt.Errorf("error parsing uri: %w", err)
	}
	if parsedURI.Scheme != "magnet" {
		return "", unexpectedSchemeError{parsedURI.Scheme}
	}
	q := parsedURI.Query()
	displayName := q.Get("dn")
	if displayName == "" {
		return "", errMissingDNQuery
	}
	return displayName, nil
}
