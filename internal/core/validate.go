package core

import (
	"net/url"
	"strings"
)

const MaxURLLen = 2048

func ValidateURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || len(raw) > MaxURLLen {
		return "", ErrInvalidURL
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return "", ErrInvalidURL
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", ErrInvalidURL
	}
	if parsed.Host == "" {
		return "", ErrInvalidURL
	}

	return parsed.String(), nil
}
