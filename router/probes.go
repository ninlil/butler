package router

import "net/http"

var (
	// Ready is the flag for the readiness-probe
	Ready bool = true

	// Healty is the flag for the liveness-probe
	Healty bool = true
)

func readyProbe(w http.ResponseWriter, r *http.Request) {
	if !Ready {
		w.WriteHeader(http.StatusNotFound)
	} else {
		w.WriteHeader(http.StatusOK)
	}
}

func healthyProbe(w http.ResponseWriter, r *http.Request) {
	if !Healty {
		w.WriteHeader(http.StatusNotFound)
	} else {
		w.WriteHeader(http.StatusOK)
	}
}
