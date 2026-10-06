package router

import (
	"crypto/tls"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/ninlil/butler/log"
)

const certCheckInterval = 30 * time.Second

// certReloader serves a cached key pair and re-reads the files when their mtime changes.
type certReloader struct {
	certFile, keyFile string
	now               func() time.Time

	mu      sync.Mutex
	cert    *tls.Certificate
	certMod time.Time
	keyMod  time.Time
	checked time.Time
}

func newCertReloader(certFile, keyFile string) (*certReloader, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("router: loading TLS key pair: %w", err)
	}
	certMod, keyMod, err := modTimes(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("router: loading TLS key pair: %w", err)
	}
	return &certReloader{
		certFile: certFile,
		keyFile:  keyFile,
		now:      time.Now,
		cert:     &cert,
		certMod:  certMod,
		keyMod:   keyMod,
		checked:  time.Now(),
	}, nil
}

func modTimes(certFile, keyFile string) (certMod, keyMod time.Time, err error) {
	ci, err := os.Stat(certFile)
	if err != nil {
		return certMod, keyMod, err
	}
	ki, err := os.Stat(keyFile)
	if err != nil {
		return certMod, keyMod, err
	}
	return ci.ModTime(), ki.ModTime(), nil
}

// GetCertificate never fails: on any reload problem the previous certificate stays in use.
func (c *certReloader) GetCertificate(*tls.ClientHelloInfo) (*tls.Certificate, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if now := c.now(); now.Sub(c.checked) >= certCheckInterval {
		c.checked = now
		c.reload()
	}
	return c.cert, nil
}

func (c *certReloader) reload() {
	certMod, keyMod, err := modTimes(c.certFile, c.keyFile)
	if err != nil {
		log.Error().Msgf("router: TLS reload: %v", err)
		return
	}
	if certMod.Equal(c.certMod) && keyMod.Equal(c.keyMod) {
		return
	}
	cert, err := tls.LoadX509KeyPair(c.certFile, c.keyFile)
	if err != nil {
		log.Error().Msgf("router: TLS reload: %v", err)
		return
	}
	c.cert, c.certMod, c.keyMod = &cert, certMod, keyMod
	log.Info().Msg("router: TLS certificate reloaded")
}

// resolveTLS validates the TLS options and builds r.tlsConfig when WithTLS was used.
func (r *Router) resolveTLS() error {
	if r.tlsCertFile != "" {
		reloader, err := newCertReloader(r.tlsCertFile, r.tlsKeyFile)
		if err != nil {
			return err
		}
		r.tlsConfig = &tls.Config{
			MinVersion:     tls.VersionTLS12,
			GetCertificate: reloader.GetCertificate,
		}
		return nil
	}

	if r.tlsConfig != nil {
		c := r.tlsConfig
		if len(c.Certificates) == 0 && c.GetCertificate == nil && c.GetConfigForClient == nil {
			return ErrorInvalidTLS
		}
	}
	return nil
}

func (r *Router) hasTLS() bool {
	return r.tlsConfig != nil
}
