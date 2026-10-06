package router

import (
	"crypto/tls"
	"net/http"
	"net/url"
)

// Option is for 'functional options' to the New and Serve-methods
type Option func(*Router) error

// WithName assigns a prefix to all router (ex "/prefix")
func WithName(name string) Option {
	return func(r *Router) error {
		r.name = name
		return nil
	}
}

// WithPrefix assigns a prefix to all router (ex "/prefix")
func WithPrefix(path string) Option {
	return func(r *Router) error {
		if err := isValidProbePath(path); err != nil {
			return err
		}
		r.prefix = path
		return nil
	}
}

// WithStrictSlash sets the StrictSlash-option on the router
func WithStrictSlash(flag bool) Option {
	return func(r *Router) error {
		r.strictSlash = flag
		return nil
	}
}

// WithPort tells what port to listen on for requests; a TLS option makes this port HTTPS
func WithPort(port int) Option {
	return func(r *Router) error {
		if port <= 0 {
			return ErrorInvalidPort
		}
		r.port = port
		r.httpsPort = 0
		return nil
	}
}

// WithPorts listens plain on httpPort and TLS on httpsPort (requires WithTLS or WithTLSConfig)
func WithPorts(httpPort, httpsPort int) Option {
	return func(r *Router) error {
		if httpPort <= 0 || httpsPort <= 0 {
			return ErrorInvalidPort
		}
		if httpPort == httpsPort {
			return ErrorPortConflict
		}
		r.port = httpPort
		r.httpsPort = httpsPort
		return nil
	}
}

// WithTLS serves TLS using the certificate and key files
func WithTLS(certFile, keyFile string) Option {
	return func(r *Router) error {
		if certFile == "" || keyFile == "" {
			return ErrorInvalidTLS
		}
		r.tlsCertFile = certFile
		r.tlsKeyFile = keyFile
		r.tlsConfig = nil
		return nil
	}
}

// WithTLSConfig serves TLS using a caller-supplied configuration
func WithTLSConfig(cfg *tls.Config) Option {
	return func(r *Router) error {
		if cfg == nil {
			return ErrorInvalidTLS
		}
		r.tlsConfig = cfg
		r.tlsCertFile = ""
		r.tlsKeyFile = ""
		return nil
	}
}

// WithHTTPSRedirect redirects plain requests to HTTPS (requires WithPorts)
func WithHTTPSRedirect() Option {
	return func(r *Router) error {
		r.httpsRedirect = true
		return nil
	}
}

// WithHealth sets the path (with leading /) that the health-probe should listen on
func WithHealth(path string) Option {
	return func(r *Router) error {
		if err := isValidProbePath(path); err != nil {
			return err
		}
		r.healthPath = path
		return nil
	}
}

// Without204 removes the automatic health-probe from the router
func Without204() Option {
	return func(r *Router) error {
		r.skip204 = true
		return nil
	}
}

// WithoutHealth removes the automatic health-probe from the router
func WithoutHealth() Option {
	return func(r *Router) error {
		r.healthPath = ""
		return nil
	}
}

// WithReady sets the path (with leading /) that the ready-probe should listen on
func WithReady(path string) Option {
	return func(r *Router) error {
		if err := isValidProbePath(path); err != nil {
			return err
		}
		r.readyPath = path
		return nil
	}
}

// WithoutReady removes the automatic ready-probe from the router
func WithoutReady() Option {
	return func(r *Router) error {
		r.readyPath = ""
		return nil
	}
}

// WithExposedErrors will send any panic-errors as request-body
func WithExposedErrors() Option {
	return func(r *Router) error {
		r.exposedErrors = true
		return nil
	}
}

// WithMiddleware adds one or more standard http middleware functions to the router chain.
// Middlewares are applied in order, after the built-in logging/ID/panic-recovery chain
// and before each route handler. The signature matches the chi/stdlib convention:
//
//	func(next http.Handler) http.Handler
func WithMiddleware(mw ...func(http.Handler) http.Handler) Option {
	return func(r *Router) error {
		r.middlewares = append(r.middlewares, mw...)
		return nil
	}
}

// Error is when a router is unable to handle to handle options or requests
type Error int

// Errors for router-options
const (
	ErrorRequireLeadingSlash Error = 1
	ErrorNotValidURL         Error = 2
	ErrorInvalidPort         Error = 3
	ErrorPortConflict        Error = 4
	ErrorTLSNotConfigured    Error = 5
	ErrorRedirectNeedsPorts  Error = 6
	ErrorInvalidTLS          Error = 7
)

func (err Error) Error() string {
	switch err {
	case ErrorRequireLeadingSlash:
		return "leading '/' is required"
	case ErrorNotValidURL:
		return "not a valid url path"
	case ErrorInvalidPort:
		return "invalid port"
	case ErrorPortConflict:
		return "http and https ports must differ"
	case ErrorTLSNotConfigured:
		return "https port requires a TLS option (WithTLS or WithTLSConfig)"
	case ErrorRedirectNeedsPorts:
		return "WithHTTPSRedirect requires WithPorts"
	case ErrorInvalidTLS:
		return "invalid TLS configuration"
	}
	return "unknown router error"
}

func isValidProbePath(path string) error {
	if path == "" {
		return nil
	}

	if path[0] != '/' {
		return ErrorRequireLeadingSlash
	}

	if _, err := url.Parse(path); err != nil {
		return ErrorNotValidURL
	}

	return nil
}
