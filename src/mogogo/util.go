package mogogo

import (
	"encoding/hex"
	"fmt"
	"labix.org/v2/mgo/bson"
	"regexp"
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

