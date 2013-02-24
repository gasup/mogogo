package mogogo

import (
	"encoding/hex"
	"fmt"
	"net/url"
	"regexp"
	"labix.org/v2/mgo/bson"
	"strconv"
	"strings"
	"unicode"
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
func typeNameToQueryName(typ string) string {
	ret := strings.ToLower(typ)
	if unicode.IsLower(rune(typ[0])) {
		ret = "-" + ret
	}
	return ret
}
func isSysQueryName(qn string) bool {
	return qn != "" && qn[0] == '-'
}

func indexOf(sa []string, s string) (index int, ok bool) {
	for i, v := range sa {
		if v == s {
			return i, true
		}
	}
	return 0, false
}
func parseObjectId(h string) (id bson.ObjectId, err error) {
	d, err := hex.DecodeString(h)
	if err != nil || len(d) != 12 {
		return bson.ObjectId(""), fmt.Errorf("id format error: %s", h)
	}
	return bson.ObjectId(d), nil
}
type URI struct {
	r           *rest
	path        []string
	QueryParams map[string]string
}

func (uri *URI) NumElem() int {
	return len(uri.path) - 1
}
func (uri *URI) Elem(index int) (val interface{}, err error) {
	if index < 0 || index >= uri.NumElem() {
		panic(fmt.Sprintln("index out of bound: %d", index))
	}
	cq := uri.r.queries[uri.path[0]]
	typ := cq.ElemType[index]
	elem := uri.path[index+1]
	switch typ {
	case "int":
		val, err = strconv.Atoi(elem)
	case "string":
		val, err = elem, nil
	case "bool":
		val, err = strconv.ParseBool(elem)
	default:
		val, err = uri.r.newWithId(typ, elem)
	}
	return
}
func (uri *URI) URLWithBase(base *url.URL) *url.URL {
	u := uri.url()
	u.Scheme = base.Scheme
	u.Host = base.Host
	return u
}
func (uri *URI) url() *url.URL {
	var u url.URL
	u.Path = "/" + strings.Join(uri.path, "/")
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
	uri.path = strings.Split(url.Path[1:], "/")
	uri.QueryParams = make(map[string]string)
	for k, v := range url.Query() {
		uri.QueryParams[k] = v[0]
	}
	return

}
