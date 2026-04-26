package router

import (
	"errors"
	"testing"
)

func TestIsValidProbePath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr error
	}{
		{"empty string", "", nil},
		{"no leading slash", "noslash", ErrorRequireLeadingSlash},
		{"valid path", "/healthz", nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := isValidProbePath(tc.path)
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("isValidProbePath(%q) = %v, want %v", tc.path, err, tc.wantErr)
			}
		})
	}
}

func TestWithPort(t *testing.T) {
	t.Run("valid port", func(t *testing.T) {
		r := &Router{}
		opt := WithPort(8080)
		if err := opt(r); err != nil {
			t.Fatalf("WithPort(8080) returned error: %v", err)
		}
		if r.port != 8080 {
			t.Errorf("expected port 8080, got %d", r.port)
		}
	})

	t.Run("zero port", func(t *testing.T) {
		r := &Router{}
		if err := WithPort(0)(r); !errors.Is(err, ErrorInvalidPort) {
			t.Errorf("WithPort(0) = %v, want ErrorInvalidPort", err)
		}
	})

	t.Run("negative port", func(t *testing.T) {
		r := &Router{}
		if err := WithPort(-1)(r); !errors.Is(err, ErrorInvalidPort) {
			t.Errorf("WithPort(-1) = %v, want ErrorInvalidPort", err)
		}
	})
}

func TestWithPrefix(t *testing.T) {
	t.Run("valid prefix", func(t *testing.T) {
		r := &Router{}
		if err := WithPrefix("/api")(r); err != nil {
			t.Fatalf("WithPrefix(\"/api\") returned error: %v", err)
		}
		if r.prefix != "/api" {
			t.Errorf("expected prefix \"/api\", got %q", r.prefix)
		}
	})

	t.Run("no leading slash", func(t *testing.T) {
		r := &Router{}
		if err := WithPrefix("api")(r); !errors.Is(err, ErrorRequireLeadingSlash) {
			t.Errorf("WithPrefix(\"api\") = %v, want ErrorRequireLeadingSlash", err)
		}
	})
}

func TestWithHealth(t *testing.T) {
	t.Run("valid path", func(t *testing.T) {
		r := &Router{}
		if err := WithHealth("/healthz")(r); err != nil {
			t.Fatalf("WithHealth(\"/healthz\") returned error: %v", err)
		}
		if r.healthPath != "/healthz" {
			t.Errorf("expected healthPath \"/healthz\", got %q", r.healthPath)
		}
	})

	t.Run("no leading slash", func(t *testing.T) {
		r := &Router{}
		if err := WithHealth("healthz")(r); !errors.Is(err, ErrorRequireLeadingSlash) {
			t.Errorf("WithHealth(\"healthz\") = %v, want ErrorRequireLeadingSlash", err)
		}
	})
}

func TestWithReady(t *testing.T) {
	t.Run("valid path", func(t *testing.T) {
		r := &Router{}
		if err := WithReady("/readyz")(r); err != nil {
			t.Fatalf("WithReady(\"/readyz\") returned error: %v", err)
		}
		if r.readyPath != "/readyz" {
			t.Errorf("expected readyPath \"/readyz\", got %q", r.readyPath)
		}
	})

	t.Run("no leading slash", func(t *testing.T) {
		r := &Router{}
		if err := WithReady("readyz")(r); !errors.Is(err, ErrorRequireLeadingSlash) {
			t.Errorf("WithReady(\"readyz\") = %v, want ErrorRequireLeadingSlash", err)
		}
	})
}
