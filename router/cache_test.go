package router

import (
	"testing"
)

func TestGetRegexp(t *testing.T) {
	t.Run("valid pattern returns non-nil regexp", func(t *testing.T) {
		cachedRegex = nil
		r, err := getRegexp("^[a-z]+")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if r == nil {
			t.Fatal("expected non-nil *regexp.Regexp")
		}
	})

	t.Run("second call returns identical pointer (cache hit)", func(t *testing.T) {
		cachedRegex = nil
		r1, err := getRegexp("^[a-z]+")
		if err != nil {
			t.Fatalf("unexpected error on first call: %v", err)
		}
		r2, err := getRegexp("^[a-z]+")
		if err != nil {
			t.Fatalf("unexpected error on second call: %v", err)
		}
		if r1 != r2 {
			t.Errorf("expected cache hit: r1 and r2 should be identical pointers")
		}
	})

	t.Run("invalid pattern returns nil and error", func(t *testing.T) {
		cachedRegex = nil
		r, err := getRegexp("[unclosed")
		if err == nil {
			t.Fatal("expected error for invalid pattern, got nil")
		}
		if r != nil {
			t.Errorf("expected nil *regexp.Regexp for invalid pattern, got %v", r)
		}
	})
}
