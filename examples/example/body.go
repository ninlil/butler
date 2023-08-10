package main

import (
	"encoding/json"
	"strings"
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

func bodyStruct(args *bodyStructArgs) string {
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

func bodyMap(args *bodyMapArgs) []byte {
	data, _ := json.MarshalIndent(args.Body, "", "  ")
	return data
}
