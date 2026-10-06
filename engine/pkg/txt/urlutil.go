package txt

import (
	"net/url"
	"strings"
)

func NormalizeURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	u, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	if u.Scheme == "" {
		u.Scheme = "http"
	}
	return u.String(), nil
}

func HostOf(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	return u.Host
}
