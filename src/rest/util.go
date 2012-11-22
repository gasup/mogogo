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
func isSysQueryName(qn string) bool {
	return qn != "" && qn[0] == '-';
}
type URI  struct {
	Path []string
	QueryParams map[string]string
}
func (uri *URI) URLWithBase(base *url.URL) *url.URL {
	u := uri.url()
	u.Scheme = base.Scheme
	u.Host = base.Host
	return u
}
func (uri *URI) url() *url.URL {
	var u url.URL
	u.Path = "/" + strings.Join(uri.Path, "/")
	vals := make(url.Values)
	for k, v := range uri.QueryParams {
		vals.Add(k, v)
	}
	u.RawQuery = vals.Encode()
	return &u
}
func (uri *URI) String() string {
	return uri.url().String()
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
