package rest

import (
	"fmt"
	"testing"
)

func TestIsQueryName1(t *testing.T) {
	testCase := []string {
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
	testCase := []string {
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
	testCase := []string {
		"https://www.google.com/%E5%88%98%E5%85%B8/%E5%88%98%E5%85%B8?q=%E5%88%98%E5%85%B8",
		"/%E5%88%98%E5%85%B8",
		"http://www.abc.com/?q=abc",
		"/?q=abc",
		"/hello?q=abc",
	}
	for _, tc := range testCase {
		_, err := URLParse(tc)
		if err != nil {
			t.Errorf("url: %s, err: %v", tc, err)
		}
	}
}
func TestParseURL2(t *testing.T) {
	uri, err := URLParse("/%E5%88%98%E5%85%B8?a=1&b=2")
	if err != nil || len(uri.Path) !=1 || uri.Path[0] != "刘典"  {
		t.Errorf("uri: %v, err: %v", uri, err)
	}
	params := uri.QueryParams
	if len(params) != 2 || params["a"] != "1" || params["b"] != "2" {
		t.Errorf("uri: %v, err: %v", uri, err)
	}
}
func TestParseURL3(t *testing.T) {
	uri, err := URLParse("/")
	if err != nil || len(uri.Path) !=1 || uri.Path[0] != ""  {
		t.Errorf("uri: %v, err: %v", uri, err)
	}
}
func TestParseURL4(t *testing.T) {
	_, err := URLParse("%E5%88%98%E5%85%B8?a=1&b=2")
	if err == nil {
		t.Fail()
	}
}
