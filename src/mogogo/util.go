package mogogo

import (
	"encoding/hex"
	"fmt"
	"labix.org/v2/mgo/bson"
	"regexp"
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
	if sa == nil {
		return -1, false
	}
	for i, v := range sa {
		if v == s {
			return i, true
		}
	}
	return -1, false
}
func parseObjectId(h string) (id bson.ObjectId, err error) {
	d, err := hex.DecodeString(h)
	if err != nil || len(d) != 12 {
		return bson.ObjectId(""), fmt.Errorf("id format error: %s", h)
	}
	return bson.ObjectId(d), nil
}

func parseParamInt(m Params, key string, def int) (ret int, err error) {
	if v, ok := m[key]; ok {
		ret, err = strconv.Atoi(v)
		if err != nil {
			msg := fmt.Sprintf("param '%s' parse error, want int, got '%s'", key, v)
			ret, err = 0, &Error{Code: BadRequest, Msg: msg, Err: err}
		}
	} else {
		ret, err = def, nil
	}
	return
}
func parseParamBool(m Params, key string, def bool) (ret bool, err error) {
	if v, ok := m[key]; ok {
		ret, err = strconv.ParseBool(v)
		if err != nil {
			msg := fmt.Sprintf("param '%s' parse error, want bool, got '%s'", key, v)
			ret, err = false, &Error{Code: BadRequest, Msg: msg, Err: err}
		}
	} else {
		ret, err = def, nil
	}
	return
}
func parseParamString(m Params, key string, def string) (ret string, err error) {
	if v, ok := m[key]; ok {
		ret, err = v, nil
	} else {
		ret, err = def, nil
	}
	return
}
func parseParamFloat(m Params, key string, def float64) (ret float64, err error) {
	if v, ok := m[key]; ok {
		ret, err = strconv.ParseFloat(v, 64)
		if err != nil {
			msg := fmt.Sprintf("param '%s' parse error, want float, got '%s'", key, v)
			ret, err = 0, &Error{Code: BadRequest, Msg: msg, Err: err}
		}
	} else {
		ret, err = def, nil
	}
	return
}
func parseParamObjectId(m Params, key string) (ret bson.ObjectId, found bool, err error) {
	if v, ok := m[key]; ok {
		ret, err = parseObjectId(v)
		if err == nil {
			found = true
		} else {
			msg := fmt.Sprintf("param '%s' parse error, want objectId, got '%s'", key, v)
			ret, found, err = "", false, &Error{Code: BadRequest, Msg: msg, Err: err}
		}
	} else {
		ret, found, err = "", false, nil
	}
	return
}
func accMapMap(m map[string]interface{}, key0, key1 string, val interface{}) {
	mv, ok := m[key0]
	var m1 map[string]interface{}
	if ok {
		m1 = mv.(map[string]interface{})
	} else {
		m1 = make(map[string]interface{})
		m[key0] = m1
	}
	m1[key1] = val
}
