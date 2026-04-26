package router

import "strings"

// convertPath translates a butler path pattern to a Go 1.22+ ServeMux pattern.
// The catch-all suffix /* becomes /{urlsuffix...}; named params {name} are unchanged.
func convertPath(path string) string {
	if strings.HasSuffix(path, "/*") {
		return path[:len(path)-2] + "/{urlsuffix...}"
	}
	return path
}

// buildPattern builds a ServeMux registration pattern from a method and path.
// A method of "*" or "" means any method (no prefix in the pattern).
func buildPattern(method, path string) string {
	p := convertPath(path)
	if method == "*" || method == "" {
		return p
	}
	return method + " " + p
}
