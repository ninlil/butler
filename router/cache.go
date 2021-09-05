package router

import (
	"regexp"
)

var (
	cachedRegex map[string]*regexp.Regexp
)

func getRegexp(p string) (r *regexp.Regexp, err error) {
	if cachedRegex == nil {
		cachedRegex = make(map[string]*regexp.Regexp)
	}
	if r, ok := cachedRegex[p]; ok {
		return r, nil
	}
	r, err = regexp.Compile(p)
	if err == nil {
		cachedRegex[p] = r
	}
	return
}
