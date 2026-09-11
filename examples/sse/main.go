package main
package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/ninlil/butler"
	"github.com/ninlil/butler/log"
	"github.com/ninlil/butler/router"
)

var routes = []router.Route{
	{Name: "events", Method: "GET", Path: "/events", Handler: events, Streaming: true},
}

func main() {
	defer butler.Cleanup(nil)

	err := router.Serve(routes, router.WithPort(10000))
	if err != nil {
		log.Fatal().Msg(err.Error())
	}

	butler.Run()
}

// events is a raw handler: Streaming: true means the router passes the real
// http.ResponseWriter, unwrapped, so Flush() reaches the client immediately.
func events(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	for i := 1; i <= 5; i++ {
		select {
		case <-r.Context().Done():
			return
		default:
		}

		fmt.Fprintf(w, "data: message %d\n\n", i)
		flusher.Flush()
		time.Sleep(time.Second)
	}
}
