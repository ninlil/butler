package main

import (
	"fmt"

	"github.com/ninlil/butler"
	"github.com/ninlil/butler/log"
	"github.com/ninlil/butler/router"
)

var routes = []router.Route{
	{Name: "test", Method: "GET", Path: "/test", Handler: testHandler},
}

func main() {
	defer butler.Cleanup(nil)

	err := router.Serve(routes, 10000)
	if err != nil {
		log.Fatal().Msg(err.Error())
	}

	butler.Run()
}

type testArgs struct {
	Email string `from:"query" json:"email" regex:"^[^@\\s]+@[^@\\s]+\\.[^@\\s]+$"`
}

func testHandler(args *testArgs) string {
	return fmt.Sprintf("Hello, %s!", args.Email)
}
