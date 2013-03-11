package mogogo

import (
	"fmt"
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

