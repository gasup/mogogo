package mogogo

import (
	"fmt"
	"net/url"
	"testing"
)

func TestIsQueryName1(t *testing.T) {
	testCase := []string{
		"",
		"abc",
		"a-b-c",
		"aa-bb-cc-123",
		"-abc",
		"-a-b-c",
		"-aa-bb-cc-123",
	}
	for _, tc := range testCase {
		ok := isQueryName(tc)
		if !ok {
			t.Error(tc)
		}
	}
}

func TestIsQueryName2(t *testing.T) {
	testCase := []string{
		"-",
		"a-b-",
		"aa--bb",
		"aa-bb-cc-",
		"-a-b-c-",
		"-aa-bb--cc-123",
	}
	for _, tc := range testCase {
		ok := isQueryName(tc)
		if ok {
			t.Error(tc)
		}
	}
}
func TestIsSysQueryName(t *testing.T) {
	ok := isSysQueryName("")
	if ok {
		t.Fail()
	}
	ok = isSysQueryName("-abc-123")
	if !ok {
		t.Fail()
	}
	ok = isSysQueryName("abc-123")
	if ok {
		t.Fail()
	}
}

func ExampleCheckQueryName() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println(r)
		}
	}()
	checkQueryName("aa-")
	//Output:'aa-' not a valid query name
}

func TestParseURL1(t *testing.T) {
	testCase := []string{
		"https://www.google.com/%E5%88%98%E5%85%B8/%E5%88%98%E5%85%B8?q=%E5%88%98%E5%85%B8",
		"/%E5%88%98%E5%85%B8",
		"http://www.abc.com/?q=abc",
		"/?q=abc",
		"/hello?q=abc",
	}
	for _, tc := range testCase {
		_, err := URIParse(tc)
		if err != nil {
			t.Errorf("url: %s, err: %v", tc, err)
		}
	}
}
func TestParseURL2(t *testing.T) {
	uri, err := URIParse("/%E5%88%98%E5%85%B8?a=1&b=2")
	if err != nil || len(uri.path) != 1 || uri.path[0] != "刘典" {
		t.Errorf("uri: %v, err: %v", uri, err)
	}
	params := uri.QueryParams
	if len(params) != 2 || params["a"] != "1" || params["b"] != "2" {
		t.Errorf("uri: %v, err: %v", uri, err)
	}
}
func TestParseURL3(t *testing.T) {
	uri, err := URIParse("/")
	if err != nil || len(uri.path) != 1 || uri.path[0] != "" {
		t.Errorf("uri: %v, err: %v", uri, err)
	}
}
func TestParseURL4(t *testing.T) {
	_, err := URIParse("%E5%88%98%E5%85%B8?a=1&b=2")
	if err == nil {
		t.Fail()
	}
}

func ExampleURI1() {
	uri := &URI{nil, []string{"你好", "hello"}, map[string]string{"a": "1"}}
	fmt.Println(uri.String())
	//Output:/%E4%BD%A0%E5%A5%BD/hello?a=1
}

func ExampleURI2() {
	u, _ := url.Parse("http://www.liudian.com/a/b")
	uri := &URI{nil, []string{"你好", "hello"}, map[string]string{"a": "1"}}
	fmt.Println(uri.URLWithBase(u))
	//Output:http://www.liudian.com/%E4%BD%A0%E5%A5%BD/hello?a=1
}
