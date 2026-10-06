package main

import (
	"net/http"
	"os"

	"github.com/ninlil/butler"
	"github.com/ninlil/butler/log"
	"github.com/ninlil/butler/router"
)

var routes = []router.Route{
	{Name: "hello", Method: "GET", Path: "/hello", Handler: hello},
}

func main() {
	defer butler.Cleanup(nil)

	certFile := env("TLS_CERT", "tls.crt")
	keyFile := env("TLS_KEY", "tls.key")

	var opts []router.Option
	switch mode := env("MODE", "both"); mode {
	case "https":
		opts = []router.Option{router.WithPort(8443), router.WithTLS(certFile, keyFile)}
	case "both":
		opts = []router.Option{router.WithPorts(10000, 10443), router.WithTLS(certFile, keyFile)}
	case "redirect":
		opts = []router.Option{router.WithPorts(10000, 10443), router.WithTLS(certFile, keyFile), router.WithHTTPSRedirect()}
	default:
		log.Fatal().Msgf("unknown MODE %q (use https, both or redirect)", mode)
	}

	if err := router.Serve(routes, opts...); err != nil {
		log.Fatal().Msg(err.Error())
	}

	butler.Run()
}

func env(name, fallback string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return fallback
}

func hello(r *http.Request) string {
	if r.TLS != nil {
		return "hello over https"
	}
	return "hello over http"
}
