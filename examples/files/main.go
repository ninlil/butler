package main

import (
	"net/http"

	"github.com/ninlil/butler"
	"github.com/ninlil/butler/bufferedresponse"
	"github.com/ninlil/butler/log"
	"github.com/ninlil/butler/router"
)

var routes = []router.Route{
	{Name: "hello", Method: "GET", Path: "/api/hello", Handler: helloHandler},
	{Name: "api-fallback", Method: "*", Path: "/api/*", Handler: notFound},
	{Name: "files", Method: "*", Path: "/*", Handler: filesHandler},
}

func helloHandler() string {
	// time.Sleep(10 * time.Second)
	return "Hello, World!"
}

func notFound() int {
	return http.StatusNotFound
}

var files = http.FileServer(http.Dir("examples/files/data"))

func filesHandler(w http.ResponseWriter, r *http.Request) int {
	w2 := bufferedresponse.Wrap(w)
	files.ServeHTTP(w2, r)
	w2.Flush()
	return w2.Status()
}

func main() {
	butler.Cleanup(nil)

	err := router.Serve(routes)
	if err != nil {
		log.Fatal().Msg(err.Error())
	}

	butler.Run()
}
