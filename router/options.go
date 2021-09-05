package router

import (
	"net/url"
)

// Option is for 'functional options' to the New and Serve-methods
type Option func(*Router) error

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

// WithStrictSlash sets the StrictSlash-option on the gorilla/mux router
func WithStrictSlash(flag bool) Option {
	return func(r *Router) error {
		r.strictSlash = flag
		return nil
	}
}

// WithPort tells what port to listen on for requests
func WithPort(port int) Option {
	return func(r *Router) error {
		if port <= 0 {
			return ErrorInvalidPort
		}
		r.port = port
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

// Error is when a router is unable to handle to handle options or requests
type Error int

// Errors for router-options
const (
	ErrorRequireLeadingSlash Error = 1
	ErrorNotValidURL         Error = 2
	ErrorInvalidPort         Error = 3
)

func (err Error) Error() string {
	switch err {
	case ErrorRequireLeadingSlash:
		return "leading '/' is required"
	case ErrorNotValidURL:
		return "not a valid url path"
	case ErrorInvalidPort:
		return "invalid port"
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
