package domain

import (
	"errors"
	"net/url"
	"strings"
	"time"
)

var (
	ErrInvalidURL = errors.New("destination must be a valid http or https URL")
	ErrInvalidCode = errors.New("short code must contain 4 to 32 letters, numbers, underscores, or hyphens")
	ErrNotFound = errors.New("link not found")
)

type Link struct {
	ID        int64     `json:"id"`
	Code      string    `json:"code"`
	URL       string    `json:"url"`
	CreatedAt time.Time `json:"created_at"`
}

func ValidateDestination(raw string) (string, error) {
	if len(raw) > 2048 {
		return "", ErrInvalidURL
	}
	u, err := url.ParseRequestURI(strings.TrimSpace(raw))
	if err != nil || u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") || u.User != nil {
		return "", ErrInvalidURL
	}
	return u.String(), nil
}

func ValidateCode(code string) bool {
	if len(code) < 4 || len(code) > 32 {
		return false
	}
	for _, r := range code {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' || r == '-') {
			return false
		}
	}
	return true
}
