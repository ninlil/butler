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
	{Name: "home", Method: "GET", Path: "/", Handler: homePage},
	{Name: "null", Method: "GET", Path: "/null", Handler: nilFunc},
	{Name: "getAll", Method: "GET", Path: "/all", Handler: returnAllArticles},
	{Name: "getRange", Method: "GET", Path: "/range", Handler: returnRangeArticles},
	{Name: "getItem", Method: "GET", Path: "/item/{index}", Handler: returnOneArticle},
	{Name: "add", Method: "POST", Path: "/add", Handler: addArticle},
	{Name: "types", Method: "*", Path: "/types", Handler: types},
	{Name: "sleep", Method: "GET", Path: "/sleep", Handler: sleep},
	{Name: "id", Method: "GET", Path: "/id", Handler: tracking},
	{Name: "sum", Method: "GET", Path: "/sum", Handler: handler},

	{Name: "body_map", Method: "POST", Path: "/body/map", Handler: bodyMap},
	{Name: "body_struct", Method: "POST", Path: "/body/struct", Handler: bodyStruct},
	{Name: "body_bytes", Method: "POST", Path: "/body/bytes", Handler: bodyBytes},
	{Name: "body_string", Method: "POST", Path: "/body/string", Handler: bodyString},
	{Name: "body_strings", Method: "POST", Path: "/body/strings", Handler: bodyStrings},
}

func main() {
	defer butler.Cleanup(nil)

	err := router.Serve(routes,
		// router.WithPrefix("/test"),
		router.WithStrictSlash(false),
		router.WithExposedErrors(),
		router.Without204(), // makes empty responses show as 200 instead of 204 "No Content"
		router.WithPort(10000))
	if err != nil {
		log.Fatal().Msg(err.Error())
	}

	butler.Run()
}

func nilFunc() error {
	return nil
}

func homePage() string {
	return "Welcome to the HomePage!"
}

func sleep() string {
	time.Sleep(10 * time.Second)
	return "slept for 10 sec"
}

// Article is a example-struct for an article
type Article struct {
	Title   string `json:"title"`
	Desc    string `json:"desc"`
	Content string `json:"content"`
}

// Articles is some data for our example
var Articles = []Article{
	{Title: "Hello", Desc: "Article Description", Content: "Article Content"},
	{Title: "Hello 2", Desc: "Article Description", Content: "Article Content"},
}

func returnAllArticles(w http.ResponseWriter) []Article {
	return Articles
}

type rangeArgs struct {
	From int `json:"from" from:"query" min:"0"`
	To   int `json:"to" from:"query" min:"0" default:"-1"`
}

func returnRangeArticles(args *rangeArgs) ([]Article, error) {
	if args.To == -1 {
		args.To = len(Articles) - 1
	}
	if args.From >= len(Articles) || args.To >= len(Articles) || args.From > args.To {
		return nil, fmt.Errorf("out-of-range: %d-%d", args.From, args.To)
	}
	return Articles[args.From : args.To+1], nil
}

type oneArgs struct {
	Index int `json:"index" from:"url" min:"0"`
}

func returnOneArticle(args *oneArgs) (*Article, int, error) {
	if args.Index >= len(Articles) {
		panic(fmt.Sprintf("error: index out-of-range: %d", args.Index))
	}
	return &Articles[args.Index], 0, nil
}

type addArgs struct {
	Body *Article `from:"body"`
}

func addArticle(args *addArgs) (int, error) {
	if args.Body == nil || args.Body.Title == "" {
		return http.StatusNotAcceptable, fmt.Errorf("error: addArticle: title missing - ignored")
	}
	Articles = append(Articles, *args.Body)
	return 0, nil
}
