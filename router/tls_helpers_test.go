package router

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// keeper stops the package-level wait-group from reaching zero between tests,
// which would otherwise fire runtime.Close() against routers of later tests.
var keeper sync.Once

// running.routers rejects duplicate names forever, so every test router gets a unique one.
var routerSeq atomic.Int64

func newTestRouter(t *testing.T, routes []Route, opts ...Option) *Router {
	t.Helper()
	keeper.Do(func() {
		running.wg = new(sync.WaitGroup)
		running.wg.Add(1)
	})
	name := fmt.Sprintf("%s_%d", strings.ReplaceAll(t.Name(), "/", "_"), routerSeq.Add(1))
	r, err := New(routes, append([]Option{WithName(name)}, opts...)...)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return r
}

type serveResult struct {
	done chan struct{}
	err  error
}

func goServeResult(r *Router) *serveResult {
	res := &serveResult{done: make(chan struct{})}
	go func() {
		res.err = r.Serve()
		close(res.done)
	}()
	return res
}

func (s *serveResult) wait(t *testing.T, d time.Duration) error {
	t.Helper()
	select {
	case <-s.done:
		return s.err
	case <-time.After(d):
		t.Fatal("Serve did not return in time")
		return nil
	}
}

func freePorts(t *testing.T, n int) []int {
	t.Helper()
	lns := make([]net.Listener, n)
	ports := make([]int, n)
	for i := range lns {
		ln, err := net.Listen("tcp", ":0")
		if err != nil {
			t.Fatalf("listen: %v", err)
		}
		lns[i] = ln
		ports[i] = ln.Addr().(*net.TCPAddr).Port
	}
	for _, ln := range lns {
		_ = ln.Close()
	}
	return ports
}

func freePort(t *testing.T) int {
	t.Helper()
	return freePorts(t, 1)[0]
}

func listenPorts(r *Router) []int {
	if r.httpsPort > 0 {
		return []int{r.port, r.httpsPort}
	}
	return []int{r.port}
}

// startRouter serves in the background, waits until every port accepts connections and stops it on cleanup.
func startRouter(t *testing.T, routes []Route, opts ...Option) (*Router, *serveResult) {
	t.Helper()
	r := newTestRouter(t, routes, opts...)
	res := goServeResult(r)
	t.Cleanup(func() {
		r.Shutdown()
		_ = res.wait(t, 5*time.Second)
	})

	deadline := time.Now().Add(5 * time.Second)
	for _, port := range listenPorts(r) {
		for {
			select {
			case <-res.done:
				t.Fatalf("Serve returned early: %v", res.err)
			default:
			}
			c, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port))
			if err == nil {
				_ = c.Close()
				break
			}
			if time.Now().After(deadline) {
				t.Fatalf("port %d not accepting connections", port)
			}
			time.Sleep(5 * time.Millisecond)
		}
	}
	return r, res
}

func plainClient() *http.Client {
	return &http.Client{
		Timeout:   5 * time.Second,
		Transport: &http.Transport{DisableKeepAlives: true},
	}
}

func tlsClient(pool *x509.CertPool) *http.Client {
	return &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			DisableKeepAlives: true,
			TLSClientConfig:   &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12},
		},
	}
}

func doGet(t *testing.T, c *http.Client, url string) (int, string) {
	t.Helper()
	resp, err := c.Get(url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(body)
}

func writeSelfSignedCert(t *testing.T) (certFile, keyFile string, pool *x509.CertPool) {
	t.Helper()
	dir := t.TempDir()
	certFile = filepath.Join(dir, "tls.crt")
	keyFile = filepath.Join(dir, "tls.key")
	return certFile, keyFile, rewriteSelfSignedCert(t, certFile, keyFile)
}

// rewriteSelfSignedCert writes a fresh key pair (new serial) to the given paths.
func rewriteSelfSignedCert(t *testing.T, certFile, keyFile string) *x509.CertPool {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generating key: %v", err)
	}

	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(time.Now().UnixNano()),
		Subject:               pkix.Name{CommonName: "localhost"},
		NotBefore:             time.Now().Add(-time.Minute),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
		DNSNames:              []string{"localhost"},
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("creating certificate: %v", err)
	}
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatalf("marshaling key: %v", err)
	}

	if err := os.WriteFile(certFile, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o600); err != nil {
		t.Fatalf("writing cert: %v", err)
	}
	if err := os.WriteFile(keyFile, pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}), 0o600); err != nil {
		t.Fatalf("writing key: %v", err)
	}

	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("parsing certificate: %v", err)
	}
	pool := x509.NewCertPool()
	pool.AddCert(cert)
	return pool
}
