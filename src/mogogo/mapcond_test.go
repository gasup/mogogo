package mogogo

import (
	"time"
	"fmt"
)

func ExampleMapCond() {
	mc := newMapCond()
	mc.Timeout = 3 * time.Second
	go func() {
		time.Sleep(1)
		m := map[string]interface{} {
			"s":"hello",
			"n":10,
			"b":false,
			"f": 3.14,
			"a": []string{"a","b"},
		}
		mc.Broadcast(m)

	}()
	m := map[string]interface{} {
		"s":"hello",
		"n":10,
		"b":false,
	}
	timeout := mc.Wait(m)
	if timeout {
		fmt.Println("timeout falied")
	}
	//Output:hello
}
