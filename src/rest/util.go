package rest

import (
	"fmt"
	"regexp"
	"net/url"
	"strings"
)

var (
	queryNameRegexp *regexp.Regexp
)

func init() {
	var err error
	queryNameRegexp, err = regexp.Compile("^(-?([a-z0-9]+-)*[a-z0-9]+|)$")
	if err != nil {
		panic(err)
	}
}

func isQueryName(s string) bool {
	return queryNameRegexp.Match([]byte(s))
}

func checkQueryName(s string) {
	if !isQueryName(s) {
		panic(fmt.Sprintf("'%s' not a valid query name", s))
	}
}

type URI  struct {
	Path []string
	QueryParams map[string]string
}
func URIParse(s string) (uri *URI, err error) {
	url, err := url.Parse(s)
	if err != nil {
		return nil, err
	}
	if url.Path[0] != '/' {
		return nil, fmt.Errorf("must absolute url. %s", s)
	}
	uri = new(URI)
	uri.Path = strings.Split(url.Path[1:], "/")
	uri.QueryParams = make(map[string]string)
	for k, v := range url.Query() {
		uri.QueryParams[k] = v[0]
	}
	return

}
