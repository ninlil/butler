package router

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestWithTLS(t *testing.T) {
	cfg := &tls.Config{GetCertificate: func(*tls.ClientHelloInfo) (*tls.Certificate, error) { return nil, nil }}

	t.Run("last wins WithTLSConfig", func(t *testing.T) {
		r := &Router{}
		if err := WithTLS("c", "k")(r); err != nil {
			t.Fatal(err)
		}
		if err := WithTLSConfig(cfg)(r); err != nil {
			t.Fatal(err)
		}
		if r.tlsCertFile != "" || r.tlsKeyFile != "" || r.tlsConfig != cfg {
			t.Errorf("unexpected state: %q %q %v", r.tlsCertFile, r.tlsKeyFile, r.tlsConfig)
		}
	})

	t.Run("last wins WithTLS", func(t *testing.T) {
		r := &Router{}
		if err := WithTLSConfig(cfg)(r); err != nil {
			t.Fatal(err)
		}
		if err := WithTLS("c", "k")(r); err != nil {
			t.Fatal(err)
		}
		if r.tlsConfig != nil || r.tlsCertFile != "c" || r.tlsKeyFile != "k" {
			t.Errorf("unexpected state: %q %q %v", r.tlsCertFile, r.tlsKeyFile, r.tlsConfig)
		}
	})

	invalid := []struct {
		name string
		opt  Option
	}{
		{"empty cert", WithTLS("", "k")},
		{"empty key", WithTLS("c", "")},
		{"nil config", WithTLSConfig(nil)},
	}
	for _, tc := range invalid {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.opt(&Router{}); !errors.Is(err, ErrorInvalidTLS) {
				t.Errorf("got %v, want ErrorInvalidTLS", err)
			}
		})
	}
}

func TestNewTLSModes(t *testing.T) {
	certFile, keyFile, _ := writeSelfSignedCert(t)
	getCert := func(*tls.ClientHelloInfo) (*tls.Certificate, error) { return nil, nil }

	t.Run("WithTLS sets TLS 1.2 minimum", func(t *testing.T) {
		r, err := New(nil, WithTLS(certFile, keyFile))
		if err != nil {
			t.Fatal(err)
		}
		if !r.hasTLS() {
			t.Error("hasTLS() = false")
		}
		if r.tlsConfig.MinVersion != tls.VersionTLS12 {
			t.Errorf("MinVersion = %x, want TLS 1.2", r.tlsConfig.MinVersion)
		}
	})

	accepted := []struct {
		name string
		opts []Option
	}{
		{"WithTLSConfig GetCertificate", []Option{WithTLSConfig(&tls.Config{GetCertificate: getCert})}},
		{"WithPorts and WithTLS", []Option{WithPorts(1, 2), WithTLS(certFile, keyFile)}},
		{"WithPorts, WithTLS and redirect", []Option{WithPorts(1, 2), WithTLS(certFile, keyFile), WithHTTPSRedirect()}},
		{"redirect before ports", []Option{WithHTTPSRedirect(), WithPorts(1, 2), WithTLS(certFile, keyFile)}},
	}
	for _, tc := range accepted {
		t.Run(tc.name, func(t *testing.T) {
			r, err := New(nil, tc.opts...)
			if err != nil {
				t.Fatal(err)
			}
			if !r.hasTLS() {
				t.Error("hasTLS() = false")
			}
		})
	}

	t.Run("plain without TLS", func(t *testing.T) {
		r, err := New(nil, WithPort(1))
		if err != nil {
			t.Fatal(err)
		}
		if r.hasTLS() {
			t.Error("hasTLS() = true")
		}
	})
}

func TestNewTLSModeErrors(t *testing.T) {
	certFile, keyFile, _ := writeSelfSignedCert(t)

	sentinel := []struct {
		name string
		opts []Option
		want error
	}{
		{"ports without TLS", []Option{WithPorts(1, 2)}, ErrorTLSNotConfigured},
		{"redirect with WithPort", []Option{WithHTTPSRedirect(), WithPort(1)}, ErrorRedirectNeedsPorts},
		{"redirect with WithPort and TLS", []Option{WithHTTPSRedirect(), WithPort(1), WithTLS(certFile, keyFile)}, ErrorRedirectNeedsPorts},
		{"empty TLS config", []Option{WithTLSConfig(&tls.Config{})}, ErrorInvalidTLS},
		{"WithPort resets WithPorts", []Option{WithPorts(1, 2), WithTLS(certFile, keyFile), WithPort(3), WithHTTPSRedirect()}, ErrorRedirectNeedsPorts},
	}
	for _, tc := range sentinel {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := New(nil, tc.opts...); !errors.Is(err, tc.want) {
				t.Errorf("got %v, want %v", err, tc.want)
			}
		})
	}
}

func TestNewTLSLoadErrors(t *testing.T) {
	certFile, keyFile, _ := writeSelfSignedCert(t)

	t.Run("missing file", func(t *testing.T) {
		_, err := New(nil, WithTLS(filepath.Join(t.TempDir(), "none.crt"), keyFile))
		if err == nil || errors.Is(err, ErrorInvalidTLS) {
			t.Errorf("got %v, want a loading error", err)
		}
	})

	t.Run("mismatched pair", func(t *testing.T) {
		_, otherKey, _ := writeSelfSignedCert(t)
		_, err := New(nil, WithTLS(certFile, otherKey))
		if err == nil || errors.Is(err, ErrorInvalidTLS) {
			t.Errorf("got %v, want a loading error", err)
		}
	})

	t.Run("garbage files", func(t *testing.T) {
		bad := filepath.Join(t.TempDir(), "bad.pem")
		if err := os.WriteFile(bad, []byte("nope"), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := New(nil, WithTLS(bad, bad)); err == nil {
			t.Error("expected error")
		}
	})
}

func TestErrorMessagesTLS(t *testing.T) {
	for _, e := range []Error{ErrorPortConflict, ErrorTLSNotConfigured, ErrorRedirectNeedsPorts, ErrorInvalidTLS} {
		if msg := e.Error(); strings.Contains(msg, "unknown") || msg == "" {
			t.Errorf("Error(%d) = %q", int(e), msg)
		}
	}
}

type reloaderFixture struct {
	*certReloader
	certFile, keyFile string
	clock             time.Time
}

func (f *reloaderFixture) advance(d time.Duration) { f.clock = f.clock.Add(d) }

func (f *reloaderFixture) get(t *testing.T) *tls.Certificate {
	t.Helper()
	cert, err := f.GetCertificate(nil)
	if err != nil {
		t.Fatalf("GetCertificate: %v", err)
	}
	return cert
}

// rotate rewrites the files and gives them an mtime newer than the seeded one.
func (f *reloaderFixture) rotate(t *testing.T) {
	t.Helper()
	rewriteSelfSignedCert(t, f.certFile, f.keyFile)
	f.touch(t)
}

func (f *reloaderFixture) touch(t *testing.T) {
	t.Helper()
	future := time.Now().Add(time.Hour)
	for _, name := range []string{f.certFile, f.keyFile} {
		if err := os.Chtimes(name, future, future); err != nil {
			t.Fatal(err)
		}
	}
}

func newReloaderFixture(t *testing.T) *reloaderFixture {
	t.Helper()
	certFile, keyFile, _ := writeSelfSignedCert(t)
	c, err := newCertReloader(certFile, keyFile)
	if err != nil {
		t.Fatal(err)
	}
	f := &reloaderFixture{certReloader: c, certFile: certFile, keyFile: keyFile, clock: time.Now()}
	c.now = func() time.Time { return f.clock }
	c.checked = f.clock
	return f
}

func serialOf(t *testing.T, cert *tls.Certificate) *big.Int {
	t.Helper()
	leaf, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		t.Fatal(err)
	}
	return leaf.SerialNumber
}

func TestCertReloaderCaches(t *testing.T) {
	f := newReloaderFixture(t)
	first := f.get(t)

	f.rotate(t)
	f.advance(certCheckInterval - time.Second)

	if f.get(t) != first {
		t.Error("certificate was reloaded inside the check interval")
	}
}

func TestCertReloaderUnchangedFilesKeepPointer(t *testing.T) {
	f := newReloaderFixture(t)
	first := f.get(t)

	f.advance(2 * certCheckInterval)

	if f.get(t) != first {
		t.Error("certificate was reloaded although the files did not change")
	}
}

func TestCertReloaderReloads(t *testing.T) {
	f := newReloaderFixture(t)
	oldSerial := serialOf(t, f.get(t))

	f.rotate(t)
	f.advance(certCheckInterval)

	if newSerial := serialOf(t, f.get(t)); newSerial.Cmp(oldSerial) == 0 {
		t.Error("certificate was not reloaded after rotation")
	}
}

func TestCertReloaderKeepsOldOnBadFiles(t *testing.T) {
	f := newReloaderFixture(t)
	first := f.get(t)

	for _, name := range []string{f.certFile, f.keyFile} {
		if err := os.WriteFile(name, []byte("garbage"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	f.touch(t)
	f.advance(certCheckInterval)

	if f.get(t) != first {
		t.Error("bad files replaced the previous certificate")
	}

	// a later valid rotation is still picked up
	f.rotate(t)
	f.advance(certCheckInterval)
	if f.get(t) == first {
		t.Error("valid rotation after a failed reload was ignored")
	}
}

func TestCertReloaderMissingFile(t *testing.T) {
	f := newReloaderFixture(t)
	first := f.get(t)

	if err := os.Remove(f.keyFile); err != nil {
		t.Fatal(err)
	}
	f.advance(certCheckInterval)

	if f.get(t) != first {
		t.Error("missing key file replaced the previous certificate")
	}
}

func TestNewTLSBadInitialPair(t *testing.T) {
	bad := filepath.Join(t.TempDir(), "bad.pem")
	if err := os.WriteFile(bad, []byte("garbage"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := New(nil, WithTLS(bad, bad)); err == nil {
		t.Error("New accepted an invalid initial key pair")
	}
}

func TestWithTLSUsesReloader(t *testing.T) {
	certFile, keyFile, _ := writeSelfSignedCert(t)
	r, err := New(nil, WithTLS(certFile, keyFile))
	if err != nil {
		t.Fatal(err)
	}
	if r.tlsConfig.GetCertificate == nil || len(r.tlsConfig.Certificates) != 0 {
		t.Error("WithTLS must serve certificates through GetCertificate only")
	}
}

func TestServeHTTPSReloadsCert(t *testing.T) {
	f := newReloaderFixture(t)
	p := freePort(t)
	startRouter(t, helloRoutes, WithPort(p),
		WithTLSConfig(&tls.Config{GetCertificate: f.GetCertificate, MinVersion: tls.VersionTLS12}))

	peerSerial := func(pool *x509.CertPool) *big.Int {
		t.Helper()
		resp, err := tlsClient(pool).Get(testURL("https", p, "/hello"))
		if err != nil {
			t.Fatalf("GET: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()
		return resp.TLS.PeerCertificates[0].SerialNumber
	}

	pool := x509.NewCertPool()
	pool.AddCert(mustParseCert(t, f.certFile))
	before := peerSerial(pool)

	pool = rewriteSelfSignedCert(t, f.certFile, f.keyFile)
	f.touch(t)
	f.advance(certCheckInterval)

	if after := peerSerial(pool); after.Cmp(before) == 0 {
		t.Error("new handshake still served the old certificate")
	}
}

func mustParseCert(t *testing.T, certFile string) *x509.Certificate {
	t.Helper()
	pair, err := tls.LoadX509KeyPair(certFile, strings.TrimSuffix(certFile, ".crt")+".key")
	if err != nil {
		t.Fatal(err)
	}
	leaf, err := x509.ParseCertificate(pair.Certificate[0])
	if err != nil {
		t.Fatal(err)
	}
	return leaf
}
