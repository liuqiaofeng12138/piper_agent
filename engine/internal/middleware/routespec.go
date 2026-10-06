package middleware

import "strings"

// Mirrors one.rewind.nio.web.route.Routes path groups (case-insensitive full path match).

const (
	noAuthRoutes = "/misc/info|/misc/overview_metrics|/metrics|/msg|/auth/login"
	initRoutes   = "/misc/info|/misc/overview_metrics|/misc/config|/misc/import|/misc/restart"
)

func pathEqualsAny(path, patterns string) bool {
	path = strings.ToLower(strings.TrimSpace(path))
	for _, p := range strings.Split(patterns, "|") {
		if path == strings.ToLower(p) {
			return true
		}
	}
	return false
}

func isMetricsPath(path string) bool {
	return strings.EqualFold(strings.TrimSpace(path), "/metrics")
}
