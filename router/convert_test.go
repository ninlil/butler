package router

import "testing"

func TestConvertPath(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{"named param unchanged", "/items/{id}", "/items/{id}"},
		{"root wildcard rewritten", "/*", "/{urlsuffix...}"},
		{"prefixed wildcard rewritten", "/api/*", "/api/{urlsuffix...}"},
		{"static path unchanged", "/health", "/health"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := convertPath(tc.path)
			if got != tc.want {
				t.Errorf("convertPath(%q) = %q, want %q", tc.path, got, tc.want)
			}
		})
	}
}

func TestBuildPattern(t *testing.T) {
	tests := []struct {
		name   string
		method string
		path   string
		want   string
	}{
		{"specific method prepended", "GET", "/items", "GET /items"},
		{"wildcard method omitted", "*", "/files/*", "/files/{urlsuffix...}"},
		{"empty method omitted", "", "/health", "/health"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := buildPattern(tc.method, tc.path)
			if got != tc.want {
				t.Errorf("buildPattern(%q, %q) = %q, want %q", tc.method, tc.path, got, tc.want)
			}
		})
	}
}
