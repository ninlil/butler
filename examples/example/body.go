package main

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/ninlil/butler/log"
)

type bodyStructData struct {
	Data string `json:"data"`
}

type bodyStructArgs struct {
	Body *bodyStructData `from:"body"`
}

type bodyByteArgs struct {
	Body []byte `from:"body"`
}

type bodyStringArgs struct {
	Body string `from:"body"`
}

type bodyStringsArgs struct {
	Body []string `from:"body"`
}

type bodyMapArgs struct {
	Body map[string]interface{} `from:"body"`
}

func bodyStruct(r *http.Request, args *bodyStructArgs) string {
	log.AddRequestFields(r, "method", "bodyStruct")
	return args.Body.Data
}

func bodyBytes(args *bodyByteArgs) string {
	return string(args.Body)
}

func bodyString(args *bodyStringArgs) string {
	return args.Body
}

func bodyStrings(args *bodyStringsArgs) string {
	return strings.Join(args.Body, ", ")
}

func bodyMap(r *http.Request, args *bodyMapArgs) []byte {
	log.AddRequestFields(r, "method", "bodyMap", "mapSize", len(args.Body))
	data, _ := json.MarshalIndent(args.Body, "", "  ")
	return data
}
