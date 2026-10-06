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

func TestWithPortResetsHTTPSPort(t *testing.T) {
	r := &Router{}
	if err := WithPorts(1, 2)(r); err != nil {
		t.Fatal(err)
	}
	if err := WithPort(3)(r); err != nil {
		t.Fatal(err)
	}
	if r.port != 3 || r.httpsPort != 0 {
		t.Errorf("port=%d httpsPort=%d, want 3 and 0", r.port, r.httpsPort)
	}
}

func TestWithPorts(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		r := &Router{}
		if err := WithPorts(1, 2)(r); err != nil {
			t.Fatal(err)
		}
		if r.port != 1 || r.httpsPort != 2 {
			t.Errorf("port=%d httpsPort=%d, want 1 and 2", r.port, r.httpsPort)
		}
	})

	t.Run("overrides WithPort", func(t *testing.T) {
		r := &Router{}
		if err := WithPort(3)(r); err != nil {
			t.Fatal(err)
		}
		if err := WithPorts(1, 2)(r); err != nil {
			t.Fatal(err)
		}
		if r.port != 1 || r.httpsPort != 2 {
			t.Errorf("port=%d httpsPort=%d, want 1 and 2", r.port, r.httpsPort)
		}
	})

	tests := []struct {
		name        string
		http, https int
		want        error
	}{
		{"zero http", 0, 1, ErrorInvalidPort},
		{"zero https", 1, 0, ErrorInvalidPort},
		{"negative http", -1, 1, ErrorInvalidPort},
		{"equal", 5, 5, ErrorPortConflict},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if err := WithPorts(tc.http, tc.https)(&Router{}); !errors.Is(err, tc.want) {
				t.Errorf("got %v, want %v", err, tc.want)
			}
		})
	}
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
