package mogogo

import (
	"fmt"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"net/url"
	"testing"
	"time"
	"reflect"
)

func TestREST1(t *testing.T) {
	ms, err := mgo.Dial("localhost")
	if err != nil {
		panic(err)
	}
	defer ms.Close()
	s := Dial(ms, "rest_test")
	fmt.Println(s)
}

type SS struct {
	Base
	S1 string
}
type UserName string
type S struct {
	Base
	S1  string
	S2  UserName
	S3  *string
	S4  *string
	B1  bool
	I1  int
	I2  int8
	I3  int64
	I4  int16
	F1  float32
	F2  float64
	ST1 SS
	A1  []string
	A2  []SS
	A3  []string
	G1  Geo
	T1  time.Time
	U1  url.URL
	U2  url.URL
}

var time1 = time.Now()
var struct1 = bson.NewObjectId()
var bson1 = bson.M{
	"_id": bson.NewObjectId(),
	"ct":  time.Now().UTC(),
	"mt":  time.Now().UTC(),
	"s1":  "Hello World",
	"s2":  "Liu Dian",
	"s3":  "Pointer",
	"b1":  true,
	"i1":  1,
	"i2":  2,
	"i3":  3,
	"i4":  4,
	"f1":  3.0,
	"f2":  6.0,
	"a1":  []interface{}{"a", "b", "c"},
	"a2":  []interface{}{bson.NewObjectId(), bson.NewObjectId(), bson.NewObjectId()},
	"g1":  []float64{1.0, 2.0},
	"t1":  time1,
	"st1": struct1,
	"u1":  "https://twitter.com/liudian",
	"u2":  "/search?q=golang",
}

func TestBsonToStruct(t *testing.T) {
	ms, err := mgo.Dial("localhost")
	if err != nil {
		panic(err)
	}
	defer ms.Close()
	session := Dial(ms, "rest_test")
	session.DefType(S{})
	rest := session.(*rest)
	var s S
	rest.bsonToStruct(bson1, &s)
	if s.S1 != "Hello World" {
		t.Error("s.S1 != 'Hello World'")
	}
	if *s.S3 != "Pointer" {
		t.Error("Pointer")
	}
	if s.S4 != nil {
		t.Error("test nil")
	}
	a1 := s.A1
	if len(a1) != 3 || a1[0] != "a" || a1[1] != "b" || a1[2] != "c" {
		t.Error("['a','b','c']")
	}
	if len(s.A3) != 0 {
		t.Error("test slice nil")
	}
	g1 := s.G1
	if g1.Lo != 1.0 || g1.La != 2.0 {
		t.Error("Geo (1.0,2.0)")
	}
	if s.T1 != time1 {
		t.Error("Time")
	}
	if s.ST1.id != struct1 {
		t.Error("Struct")
	}
	if s.U1.String() != "https://twitter.com/liudian" {
		t.Error("URL")
	}
}
func TestStructToBson(t *testing.T) {
	ms, err := mgo.Dial("localhost")
	if err != nil {
		panic(err)
	}
	defer ms.Close()
	session := Dial(ms, "rest_test")
	session.DefType(S{})
	rest := session.(*rest)
	var s S
	rest.bsonToStruct(bson1, &s)
	bb := rest.structToBson(&s)
	if bb["s1"].(string) != "Hello World" {
		t.Error("structToBson")
	}
}

var map1 = map[string]interface{}{
	"id": bson.NewObjectId().Hex(),
	"ct": time.Now().UTC().Format(time.RFC3339),
	"mt": time.Now().UTC().Format(time.RFC3339),
	"s1": "Hello World",
	"s2": "Liu Dian",
	"s3": "Pointer",
	"b1": true,
	"i1": 1,
	"i2": 2,
	"i3": 3,
	"i4": 4.0,
	"f1": 3.0,
	"f2": 6.0,
	"a1": []interface{}{"a", "b", "c"},
	"a2": []interface{}{
		map[string]interface{}{"id": bson.NewObjectId().Hex()},
		map[string]interface{}{"id": bson.NewObjectId().Hex()},
		map[string]interface{}{"id": bson.NewObjectId().Hex()},
	},
	"g1":  map[string]interface{}{"lon": float64(1.0), "lat": float64(2.0)},
	"t1":  time1.Format(time.RFC3339),
	"st1": map[string]interface{}{"id": struct1.Hex()},
	"u1":  "https://twitter.com/liudian",
	"u2":  "/search?q=golang",
}
var baseURL1, _ = url.Parse("http://abc.com/efg")

func TestMapToStruct(t *testing.T) {
	ms, err := mgo.Dial("localhost")
	if err != nil {
		panic(err)
	}
	defer ms.Close()
	session := Dial(ms, "rest_test")
	session.DefType(S{})
	rest := session.(*rest)
	var s S
	err = rest.mapToStruct(map1, &s, baseURL1)
	if err != nil {
		t.Error(err)
		return
	}
	if s.S1 != "Hello World" {
		t.Error("s.S1 != 'Hello World'")
	}
	if *s.S3 != "Pointer" {
		t.Error("Pointer")
	}
	if s.S4 != nil {
		t.Error("test nil")
	}
	a1 := s.A1
	if len(a1) != 3 || a1[0] != "a" || a1[1] != "b" || a1[2] != "c" {
		t.Error("['a','b','c']")
	}
	if len(s.A3) != 0 {
		t.Error("test slice nil")
	}
	g1 := s.G1
	if g1.Lo != 1.0 || g1.La != 2.0 {
		t.Error("Geo (1.0,2.0)")
	}
	if s.T1.Unix() != time1.Unix() {
		t.Errorf("%v != %v", s.T1, time1)
	}
	if s.ST1.id != struct1 {
		t.Error("Struct")
	}
	if s.U1.String() != "https://twitter.com/liudian" {
		t.Error("URL")
	}
}

func ExampleMapToStruct1() {
	ms, err := mgo.Dial("localhost")
	if err != nil {
		panic(err)
	}
	defer ms.Close()
	session := Dial(ms, "rest_test")
	session.DefType(S{})
	rest := session.(*rest)
	var s struct {
		Base
		F int
	}
	err = rest.mapToStruct(map[string]interface{}{"f": 1.1}, &s, baseURL1)
	fmt.Println(err)
	//Output:field 'f' want type 'int' but 'float64'
}

func ExampleMapToStruct2() {
	ms, err := mgo.Dial("localhost")
	if err != nil {
		panic(err)
	}
	defer ms.Close()
	session := Dial(ms, "rest_test")
	session.DefType(S{})
	rest := session.(*rest)
	var s struct {
		Base
		F []int
	}
	err = rest.mapToStruct(map[string]interface{}{"f": []int{1, 2, 3}}, &s, baseURL1)
	fmt.Println(s.F)
	//Output:[1 2 3]
}
func ExampleMapToStruct3() {
	ms, err := mgo.Dial("localhost")
	if err != nil {
		panic(err)
	}
	defer ms.Close()
	session := Dial(ms, "rest_test")
	session.DefType(S{})
	rest := session.(*rest)
	var s struct {
		Base
		F int
	}
	err = rest.mapToStruct(map[string]interface{}{"f": uint(1)}, &s, baseURL1)
	fmt.Println(err)
	//Output:field 'f' want type 'int' but 'uint'
}
func ExampleMapToStruct4() {
	ms, err := mgo.Dial("localhost")
	if err != nil {
		panic(err)
	}
	defer ms.Close()
	session := Dial(ms, "rest_test")
	session.DefType(S{})
	rest := session.(*rest)
	var s struct {
		Base
		U1 url.URL
		U2 url.URL
	}
	u1 := "http://efg.com/abc?a=b"
	u2 := "http://abc.com/xyz?c=d"
	err = rest.mapToStruct(map[string]interface{}{"u1": u1, "u2": u2}, &s, baseURL1)
	fmt.Println(s.U1.String())
	fmt.Println(s.U2.String())
	//Output:http://efg.com/abc?a=b
	///xyz?c=d
}
func ExampleStructToMap() {
	id1 := bson.ObjectIdHex("513063ef69ca944b1000000a")
	tm1, _ := time.Parse(time.RFC3339, "2013-03-01T08:16:47Z")
	ms, err := mgo.Dial("localhost")
	if err != nil {
		panic(err)
	}
	defer ms.Close()
	session := Dial(ms, "rest_test")
	session.DefType(S{})
	rest := session.(*rest)
	var s struct {
		Base
		F  int
		S  SS
		U1 url.URL
		U2 url.URL
	}
	s.id = id1
	s.mt = tm1
	s.ct = tm1
	s.t = "S"
	s.loaded = true
	s.F = 100
	s.S.id = id1
	s.S.t = "SS"
	u1, _ := url.Parse("http://efg.com/abc?a=b")
	u2, _ := url.Parse("/xyz?c=d")
	s.U1 = *u1
	s.U2 = *u2
	s.S.loaded = true
	m := rest.structToMap(&s, baseURL1)
	fmt.Println(m["self"])
	fmt.Println(m["s"].(map[string]interface{})["self"])
	fmt.Println(m["u1"])
	fmt.Println(m["u2"])
	//Output:http://abc.com/s/513063ef69ca944b1000000a
	//http://abc.com/ss/513063ef69ca944b1000000a
	//http://efg.com/abc?a=b
	//http://abc.com/xyz?c=d
}

func ExampleFieldQueryPost1() {
	ms, err := mgo.Dial("localhost")
	if err != nil {
		panic(err)
	}
	defer ms.Close()
	//err = ms.DB("rest_test").DropDatabase()
	//if err != nil {
	//	panic(err)
	//}
	s := Dial(ms, "rest_test")
	s.DefType(SS{})
	s.Def("test-ss", FieldQuery{
		Type: "SS",
		Allow: POST,
	})
	ctx := NewContext()
	uri, err := URIParse("/test-ss")
	if err != nil {
		panic(err)
	}
	data := SS{S1:"Hello World"}
	r, err := s.R(uri, ctx)
	if err != nil {
		panic(err)
	}
	resp, err := r.Post(&data)
	if err != nil {
		panic(err)
	}
	fmt.Println(resp.(*SS).S1)
	//Output:Hello World
}
type SSS struct {
	Base
	S1 string
	I1 *int
	B1 bool
	S2 SS
	S3 *SS
}
func ExampleFieldQueryPost2() {
	ms, err := mgo.Dial("localhost")
	if err != nil {
		panic(err)
	}
	defer ms.Close()
	//err = ms.DB("rest_test").DropDatabase()
	//if err != nil {
	//	panic(err)
	//}
	s := Dial(ms, "rest_test")
	rest := s.(*rest)
	s.DefType(SS{})
	s.DefType(SSS{})
	s.Def("test-sss", FieldQuery{
		Type: "SSS",
		Allow: POST,
		Fields: []string{"S1", "I1"},
		ContextRef: map[string]string{
			"B1":"CB1",
			"S2":"CS2",
			"S3":"CS3",
		},
	})
	ctx := NewContext()
	ctx.Set("CB1", true)
	ss, err := rest.newWithObjectId(reflect.TypeOf(SS{}), bson.ObjectIdHex("513b090869ca940ef500000b"))
	if err != nil {
		panic(err)
	}
	ctx.Set("CS2", ss)
	ctx.Set("CS3", ss)
	uri, err := URIParse("/test-sss/hello-world/123")
	if err != nil {
		panic(err)
	}
	data := SSS{S1:"Hello World"}
	r, err := s.R(uri, ctx)
	if err != nil {
		panic(err)
	}
	resp, err := r.Post(&data)
	if err != nil {
		panic(err)
	}

	fmt.Println(resp.(*SSS).S1)
	fmt.Println(*resp.(*SSS).I1)
	fmt.Println(resp.(*SSS).B1)
	fmt.Println(resp.(*SSS).S2.id.Hex())
	fmt.Println(resp.(*SSS).S3.id.Hex())
	//Output:hello-world
	//123
	//true
	//513b090869ca940ef500000b
	//513b090869ca940ef500000b
}

func ExampleFieldQueryDelete1() {
	ms, err := mgo.Dial("localhost")
	if err != nil {
		panic(err)
	}
	defer ms.Close()
	//err = ms.DB("rest_test").DropDatabase()
	//if err != nil {
	//	panic(err)
	//}
	s := Dial(ms, "rest_test")
	rest := s.(*rest)
	s.DefType(SS{})
	s.DefType(SSS{})
	s.Def("test-sss", FieldQuery{
		Type: "SSS",
		Allow: POST | DELETE,
		Fields: []string{"S1", "I1"},
		ContextRef: map[string]string{
			"B1":"CB1",
			"S2":"CS2",
			"S3":"CS3",
		},
	})
	ctx := NewContext()
	ctx.Set("CB1", true)
	ss, err := rest.newWithObjectId(reflect.TypeOf(SS{}), bson.ObjectIdHex("513b090869ca940ef500000b"))
	if err != nil {
		panic(err)
	}
	ctx.Set("CS2", ss)
	ctx.Set("CS3", ss)
	uri, err := URIParse("/test-sss/hello-world/456")
	if err != nil {
		panic(err)
	}
	data := SSS{S1:"Hello World"}
	r, err := s.R(uri, ctx)
	if err != nil {
		panic(err)
	}
	resp, err := r.Post(&data)
	if err != nil {
		panic(err)
	}
	resp, err = r.Delete()
	fmt.Println(resp, err)
	//Output:<nil> <nil>
}
