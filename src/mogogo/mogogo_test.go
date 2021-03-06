package mogogo

import (
	"fmt"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"net/url"
	"reflect"
	"testing"
	"time"
)

func TestParseURL1(t *testing.T) {
	testCase := []string{
		"https://www.google.com/%E5%88%98%E5%85%B8/%E5%88%98%E5%85%B8?q=%E5%88%98%E5%85%B8",
		"/%E5%88%98%E5%85%B8",
		"http://www.abc.com/?q=abc",
		"/?q=abc",
		"/hello?q=abc",
	}
	for _, tc := range testCase {
		_, err := ResIdParse(tc)
		if err != nil {
			t.Errorf("url: %s, err: %v", tc, err)
		}
	}
}
func TestParseURL2(t *testing.T) {
	uri, err := ResIdParse("/%E5%88%98%E5%85%B8?a=1&b=2")
	if err != nil || len(uri.path) != 1 || uri.path[0] != "刘典" {
		t.Errorf("uri: %v, err: %v", uri, err)
	}
	params := uri.Params
	if len(params) != 2 || params["a"] != "1" || params["b"] != "2" {
		t.Errorf("uri: %v, err: %v", uri, err)
	}
}
func TestParseURL3(t *testing.T) {
	uri, err := ResIdParse("/")
	if err != nil || len(uri.path) != 1 || uri.path[0] != "" {
		t.Errorf("uri: %v, err: %v", uri, err)
	}
}
func TestParseURL4(t *testing.T) {
	_, err := ResIdParse("%E5%88%98%E5%85%B8?a=1&b=2")
	if err == nil {
		t.Fail()
	}
}

func ExampleResId1() {
	uri := &ResId{nil, []string{"你好", "hello"}, map[string]string{"a": "1"}}
	fmt.Println(uri.String())
	//Output:/%E4%BD%A0%E5%A5%BD/hello?a=1
}

func ExampleResId2() {
	u, _ := url.Parse("http://www.liudian.com/a/b")
	uri := &ResId{nil, []string{"你好", "hello"}, map[string]string{"a": "1"}}
	fmt.Println(uri.URLWithBase(u))
	//Output:http://www.liudian.com/%E4%BD%A0%E5%A5%BD/hello?a=1
}
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
type UserNameV string

func (un UserNameV) Verify() (ok bool, msg string) {
	return false, "too_short"
}

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
func ExampleMapToStruct5() {
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
		F UserNameV
	}
	err = rest.mapToStruct(map[string]interface{}{"f": "liudian"}, &s, baseURL1)
	fmt.Println(err.(*Error).Fields)
	//Output:map[F:too_short]
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
	fmt.Println(m["s"].(map[string]interface{})["href"])
	fmt.Println(m["u1"])
	fmt.Println(m["u2"])
	//Output:http://abc.com/s/513063ef69ca944b1000000a
	//http://abc.com/ss/513063ef69ca944b1000000a
	//http://efg.com/abc?a=b
	//http://abc.com/xyz?c=d
}

func ExampleFieldResourcePost1() {
	ms, err := mgo.Dial("localhost")
	if err != nil {
		panic(err)
	}
	defer ms.Close()
	err = ms.DB("rest_test").C("ss").DropCollection()
	if err != nil {
		panic(err)
	}
	s := Dial(ms, "rest_test")
	s.DefType(SS{})
	s.DefRes("test-ss", FieldResource{
		Type:  "SS",
		Allow: POST,
	})
	ctx := s.NewContext()
	defer ctx.Close()
	uri, err := ResIdParse("/test-ss")
	if err != nil {
		panic(err)
	}
	data := SS{S1: "Hello World"}
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

func ExampleFieldResourcePost2() {
	ms, err := mgo.Dial("localhost")
	if err != nil {
		panic(err)
	}
	defer ms.Close()
	err = ms.DB("rest_test").C("sss").DropCollection()
	if err != nil {
		panic(err)
	}
	s := Dial(ms, "rest_test")
	rest := s.(*rest)
	s.DefType(SS{})
	s.DefType(SSS{})
	s.DefRes("test-sss", FieldResource{
		Type:   "SSS",
		Allow:  POST,
		Fields: []string{"S1", "I1"},
		ContextRef: map[string]string{
			"B1": "CB1",
			"S2": "CS2",
			"S3": "CS3",
		},
	})
	ctx := s.NewContext()
	defer ctx.Close()
	ctx.Set("CB1", true)
	ss, err := rest.newWithObjectId(reflect.TypeOf(SS{}), bson.ObjectIdHex("513b090869ca940ef500000b"))
	if err != nil {
		panic(err)
	}
	ctx.Set("CS2", ss)
	ctx.Set("CS3", ss)
	uri, err := ResIdParse("/test-sss/hello-world/123")
	if err != nil {
		panic(err)
	}
	data := SSS{S1: "Hello World"}
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

func ExampleFieldResourceDelete1() {
	ms, err := mgo.Dial("localhost")
	if err != nil {
		panic(err)
	}
	defer ms.Close()
	err = ms.DB("rest_test").C("sss").DropCollection()
	if err != nil {
		panic(err)
	}
	s := Dial(ms, "rest_test")
	rest := s.(*rest)
	s.DefType(SS{})
	s.DefType(SSS{})
	s.DefRes("test-sss", FieldResource{
		Type:   "SSS",
		Allow:  POST | DELETE,
		Fields: []string{"S1", "I1"},
		ContextRef: map[string]string{
			"B1": "CB1",
			"S2": "CS2",
			"S3": "CS3",
		},
	})
	ctx := s.NewContext()
	defer ctx.Close()
	ctx.Set("CB1", true)
	ss, err := rest.newWithObjectId(reflect.TypeOf(SS{}), bson.ObjectIdHex("513b090869ca940ef500000b"))
	if err != nil {
		panic(err)
	}
	ctx.Set("CS2", ss)
	ctx.Set("CS3", ss)
	uri, err := ResIdParse("/test-sss/hello-world/456")
	if err != nil {
		panic(err)
	}
	data := SSS{S1: "Hello World"}
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
func ExampleFieldResourcePut1() {
	ms, err := mgo.Dial("localhost")
	if err != nil {
		panic(err)
	}
	defer ms.Close()
	err = ms.DB("rest_test").C("sss").DropCollection()
	if err != nil {
		panic(err)
	}
	s := Dial(ms, "rest_test")
	rest := s.(*rest)
	s.DefType(SS{})
	s.DefType(SSS{})
	s.DefRes("test-sss", FieldResource{
		Type:   "SSS",
		Allow:  PUT | DELETE,
		Fields: []string{"S1", "I1"},
		ContextRef: map[string]string{
			"B1": "CB1",
			"S2": "CS2",
			"S3": "CS3",
		},
		Unique: true,
	})
	ctx := s.NewContext()
	defer ctx.Close()
	ctx.Set("CB1", true)
	ss, err := rest.newWithObjectId(reflect.TypeOf(SS{}), bson.ObjectIdHex("513b090869ca940ef500000b"))
	if err != nil {
		panic(err)
	}
	ctx.Set("CS2", ss)
	ctx.Set("CS3", ss)
	uri, err := ResIdParse("/test-sss/hello-world/456")
	if err != nil {
		panic(err)
	}
	data := SSS{S1: "Hello World"}
	r, err := s.R(uri, ctx)
	if err != nil {
		panic(err)
	}
	resp, err := r.Put(&data)
	if err != nil {
		panic(err)
	}
	resp, err = r.Put(resp)
	if err != nil {
		panic(err)
	}
	resp, err = r.Delete()
	fmt.Println(resp, err)
	//Output:<nil> <nil>
}
func ExampleFieldResourceGet1() {
	ms, err := mgo.Dial("localhost")
	if err != nil {
		panic(err)
	}
	defer ms.Close()
	err = ms.DB("rest_test").C("ss").DropCollection()
	if err != nil {
		panic(err)
	}
	s := Dial(ms, "rest_test")
	s.DefType(SS{})
	s.DefRes("test-ss", FieldResource{
		Type:  "SS",
		Allow: POST,
	})
	s.Before(POST, "test-ss", func(req *Req, ctx *Context) (goOn bool, resp interface{}, err error) {
		fmt.Println("Before Post", req.Body.(*SS).S1)
		return true, nil, nil
	})
	s.After(POST, "test-ss", func(req *Req, ctx *Context, resp interface{}, err error) (goOn bool, newResp interface{}, newErr error) {
		fmt.Println("After Post", req.Body.(*SS).S1)
		return true, nil, nil
	})
	ctx := s.NewContext()
	defer ctx.Close()
	uri, err := ResIdParse("/test-ss")
	if err != nil {
		panic(err)
	}
	data := SS{S1: "Hello World"}
	r, err := s.R(uri, ctx)
	if err != nil {
		panic(err)
	}
	resp, err := r.Post(&data)
	if err != nil {
		panic(err)
	}
	r, err = s.R(data.Self(), ctx)
	resp, err = r.Get()
	if err != nil {
		panic(err)
	}
	fmt.Println(resp.(*SS).S1)
	//Output:Before Post Hello World
	//After Post Hello World
	//Hello World
}
func ExampleFieldResourceGet2() {
	ms, err := mgo.Dial("localhost")
	if err != nil {
		panic(err)
	}
	defer ms.Close()
	err = ms.DB("rest_test").C("ss").DropCollection()
	if err != nil {
		panic(err)
	}
	s := Dial(ms, "rest_test")
	s.DefType(SS{})
	s.DefRes("test-ss", FieldResource{
		Type:  "SS",
		Allow: GET | POST,
	})
	ctx := s.NewContext()
	defer ctx.Close()
	uri, err := ResIdParse("/test-ss")
	if err != nil {
		panic(err)
	}
	r, err := s.R(uri, ctx)
	if err != nil {
		panic(err)
	}
	for i := 0; i < 5; i++ {
		data := SS{S1: fmt.Sprintf("Hello %d", i)}
		_, err := r.Post(&data)
		if err != nil {
			panic(err)
		}
	}
	resp, err := r.Get()
	if err != nil {
		panic(err)
	}
	iter := resp.(Iter)
	n := iter.Count()
	fmt.Println(n)
	for {
		resp, ok := iter.Next()
		if !ok {
			break
		}
		ss := resp.(*SS)
		fmt.Println(ss.S1)
	}
	var s1set []string
	iter.Extract("S1", &s1set)
	fmt.Println(len(s1set))
	//Output:5
	//Hello 4
	//Hello 3
	//Hello 2
	//Hello 1
	//Hello 0
	//5

}
func ExampleBaseLoad() {
	ms, err := mgo.Dial("localhost")
	if err != nil {
		panic(err)
	}
	defer ms.Close()
	err = ms.DB("rest_test").C("ss").DropCollection()
	if err != nil {
		panic(err)
	}
	s := Dial(ms, "rest_test")
	rest := s.(*rest)
	s.DefType(SS{})
	s.DefRes("test-ss", FieldResource{
		Type:  "SS",
		Allow: GET | POST,
	})
	ctx := s.NewContext()
	defer ctx.Close()
	uri, err := ResIdParse("/test-ss")
	if err != nil {
		panic(err)
	}
	r, err := s.R(uri, ctx)
	if err != nil {
		panic(err)
	}
	data := SS{S1: "Hello World"}
	resp, err := r.Post(&data)
	if err != nil {
		panic(err)
	}
	ss := rest.newStruct("SS").(*SS)
	ss.id = resp.(*SS).id
	ok := ss.Load(ctx)
	if !ok {
		panic("not found")
	}
	fmt.Println(ss.S1)
	//Output:Hello World
}

type SSChild struct {
	Base
	P  *SS
	S1 string
	B1 bool
}

func ExampleBind() {
	ms, err := mgo.Dial("localhost")
	if err != nil {
		panic(err)
	}
	defer ms.Close()
	err = ms.DB("rest_test").C("ss").DropCollection()
	err = ms.DB("rest_test").C("sschild").DropCollection()
	if err != nil {
		panic(err)
	}
	s := Dial(ms, "rest_test")
	s.DefType(SS{})
	s.DefType(SSChild{})
	s.DefRes("test-ss", FieldResource{
		Type:  "SS",
		Allow: GET | POST,
	})
	s.DefRes("ss-child", FieldResource{
		Type:   "SSChild",
		Allow:  GET | POST,
		Fields: []string{"P", "B1"},
	})
	s.Bind("child", "SS", "ss-child", []interface{}{F("Id"), true})

	ctx := s.NewContext()
	defer ctx.Close()
	uri := NewResId("test-ss")
	if err != nil {
		panic(err)
	}
	r, err := s.R(uri, ctx)
	if err != nil {
		panic(err)
	}
	data := SS{S1: "Hello World"}
	resp, err := r.Post(&data)
	if err != nil {
		panic(err)
	}
	ss := resp.(*SS)
	sschild := &SSChild{S1: "Hello Child"}
	resp, err = ss.R("child", ctx).Post(sschild)
	resp, err = ss.R("child", ctx).Post(sschild)
	if err != nil {
		panic(err)
	}
	sschild = resp.(*SSChild)
	fmt.Println(sschild.S1)
	fmt.Println(sschild.B1)
	fmt.Println(ss.id == sschild.P.id)
	resp, err = ss.R("child", ctx).Get()
	if err != nil {
		panic(err)
	}
	iter := resp.(Iter)
	fmt.Println(iter.Count())
	//Output:Hello Child
	//true
	//true
	//2
}

func ExampleToMgoSelector() {
	ms, err := mgo.Dial("localhost")
	if err != nil {
		panic(err)
	}
	defer ms.Close()
	session := Dial(ms, "rest_test")
	session.DefType(S{})
	rest := session.(*rest)
	sq := SelectorResource{Type: "S"}
	h := newSQHandler(rest, &sq)
	tm1, _ := time.Parse(time.RFC3339, "2013-03-01T08:16:47Z")
	s, _ := rest.newWithId("S", "513063ef69ca944b1000000a")
	s1 := s.(*S)
	m := M{
		"S1": "Hello",
		"Id": s1,
		"A1": []interface{}{"a", "b", "c"},
		"A2": M{"$in": []*S{s1, s1, s1}},
		"T1": tm1,
		//db.places.find( { loc: { $within: { $centerSphere: [ [ -74, 40.74 ] , 100 / 6378.137 ] } } } )
		"G1":  M{"$within": M{"$centerSphere": A{Geo{La: 1.2, Lo: 3.4}, 100 / 6378.137}}},
		"$or": A{M{"S1": "Bye"}},
	}
	sel := h.toMgoSelector(m)
	fmt.Println(sel["s1"])
	fmt.Println(sel["_id"])
	fmt.Println(sel["a1"])
	fmt.Println(sel["a2"])
	fmt.Println(sel["t1"])
	fmt.Println(sel["g1"])
	fmt.Println(sel["$or"])
	//Output:Hello
	//ObjectIdHex("513063ef69ca944b1000000a")
	//[a b c]
	//map[$in:[ObjectIdHex("513063ef69ca944b1000000a") ObjectIdHex("513063ef69ca944b1000000a") ObjectIdHex("513063ef69ca944b1000000a")]]
	//2013-03-01 08:16:47 +0000 UTC
	//map[$within:map[$centerSphere:[[3.4 1.2] 0.01567855942887398]]]
	//[map[s1:Bye]]
}
func ExampleSelectorResource() {
	ms, err := mgo.Dial("localhost")
	if err != nil {
		panic(err)
	}
	defer ms.Close()
	err = ms.DB("rest_test").C("ss").DropCollection()
	if err != nil {
		panic(err)
	}
	s := Dial(ms, "rest_test")
	s.DefType(SS{})
	s.DefRes("test-ss", FieldResource{
		Type:  "SS",
		Allow: GET | POST,
	})
	s.DefRes("test-ss-sel", SelectorResource{
		Type: "SS",
		SelectorFunc: func(req *Req, ctx *Context) (M, error) {
			return M{
				"S1": M{"$gt": "Hello 2"},
			}, nil
		},
		SortFields: []string{"S1"},
	})
	ctx := s.NewContext()
	defer ctx.Close()
	uri, err := ResIdParse("/test-ss")
	if err != nil {
		panic(err)
	}
	r, err := s.R(uri, ctx)
	if err != nil {
		panic(err)
	}
	for i := 0; i < 5; i++ {
		data := SS{S1: fmt.Sprintf("Hello %d", i)}
		_, err := r.Post(&data)
		if err != nil {
			panic(err)
		}
	}
	uri, err = ResIdParse("/test-ss-sel")
	if err != nil {
		panic(err)
	}
	r, err = s.R(uri, ctx)
	if err != nil {
		panic(err)
	}
	resp, err := r.Get()
	if err != nil {
		panic(err)
	}
	iter := resp.(Iter)
	n := iter.Count()
	fmt.Println(n)
	for {
		resp, ok := iter.Next()
		if !ok {
			break
		}
		ss := resp.(*SS)
		fmt.Println(ss.S1)
	}
	//Output:2
	//Hello 3
	//Hello 4
}
func ExampleFieldResourceGetSlice1() {
	ms, err := mgo.Dial("localhost")
	if err != nil {
		panic(err)
	}
	defer ms.Close()
	err = ms.DB("rest_test").C("ss").DropCollection()
	if err != nil {
		panic(err)
	}
	s := Dial(ms, "rest_test")
	s.DefType(SS{})
	s.DefRes("test-ss", FieldResource{
		Type:       "SS",
		Allow:      GET | POST,
		SortFields: []string{"S1"},
		Count:      true,
		Limit:      4,
	})
	ctx := s.NewContext()
	defer ctx.Close()
	uri, err := ResIdParse("/test-ss?n=2")
	if err != nil {
		panic(err)
	}
	r, err := s.R(uri, ctx)
	if err != nil {
		panic(err)
	}
	for i := 0; i < 5; i++ {
		data := SS{S1: fmt.Sprintf("Hello %d", i)}
		_, err := r.Post(&data)
		if err != nil {
			panic(err)
		}
	}
	resp, err := r.Get()
	if err != nil {
		panic(err)
	}
	iter := resp.(Iter)
	slice, err := iter.Slice()
	if err != nil {
		panic(err)
	}
	fmt.Println(slice.Count())
	fmt.Println(slice.More())
	fmt.Println(slice.HasPrev())
	for _, i := range slice.Items() {
		ss := i.(*SS)
		fmt.Println(ss.S1)
	}
	r, err = s.R(slice.Next(), ctx)
	resp, err = r.Get()
	iter = resp.(Iter)
	slice, err = iter.Slice()
	for _, i := range slice.Items() {
		ss := i.(*SS)
		fmt.Println(ss.S1)
	}
	r, err = s.R(slice.Prev(), ctx)
	resp, err = r.Get()
	iter = resp.(Iter)
	slice, err = iter.Slice()
	for _, i := range slice.Items() {
		ss := i.(*SS)
		fmt.Println(ss.S1)
	}

	//Output:4
	//true
	//false
	//Hello 0
	//Hello 1
	//Hello 2
	//Hello 3
	//Hello 0
	//Hello 1
}
func ExampleFieldResourceGetSlice2() {
	ms, err := mgo.Dial("localhost")
	if err != nil {
		panic(err)
	}
	defer ms.Close()
	err = ms.DB("rest_test").C("ss").DropCollection()
	if err != nil {
		panic(err)
	}
	s := Dial(ms, "rest_test")
	s.DefType(SS{})
	s.DefRes("test-ss", FieldResource{
		Type:  "SS",
		Allow: GET | POST,
		Count: true,
		Limit: 4,
	})
	ctx := s.NewContext()
	defer ctx.Close()
	uri, err := ResIdParse("/test-ss?n=2")
	if err != nil {
		panic(err)
	}
	r, err := s.R(uri, ctx)
	if err != nil {
		panic(err)
	}
	for i := 0; i < 5; i++ {
		data := SS{S1: fmt.Sprintf("Hello %d", i)}
		_, err := r.Post(&data)
		if err != nil {
			panic(err)
		}
	}
	resp, err := r.Get()
	if err != nil {
		panic(err)
	}
	iter := resp.(Iter)
	slice, err := iter.Slice()
	if err != nil {
		panic(err)
	}
	fmt.Println(slice.Count())
	fmt.Println(slice.More())
	fmt.Println(slice.HasPrev())
	for _, i := range slice.Items() {
		ss := i.(*SS)
		fmt.Println(ss.S1)
	}
	r, err = s.R(slice.Next(), ctx)
	resp, err = r.Get()
	iter = resp.(Iter)
	slice, err = iter.Slice()
	for _, i := range slice.Items() {
		ss := i.(*SS)
		fmt.Println(ss.S1)
	}
	r, err = s.R(slice.Prev(), ctx)
	resp, err = r.Get()
	iter = resp.(Iter)
	slice, err = iter.Slice()
	for _, i := range slice.Items() {
		ss := i.(*SS)
		fmt.Println(ss.S1)
	}

	//Output:4
	//true
	//true
	//Hello 4
	//Hello 3
	//Hello 2
	//Hello 1
	//Hello 4
	//Hello 3
}
func ExampleToMgoUpdater() {
	ms, err := mgo.Dial("localhost")
	if err != nil {
		panic(err)
	}
	defer ms.Close()
	session := Dial(ms, "rest_test")
	session.DefType(S{})
	session.DefType(SS{})
	rest := session.(*rest)
	fq := FieldResource{Type: "S", PatchFields: []string{"S1", "ST1", "A1", "A2", "I1"}}
	h := newFQHandler(rest, &fq)
	s, _ := rest.newWithId("SS", "513063ef69ca944b1000000a")
	s1 := s.(*SS)
	m := M{
		"Set": M{
			"S1":  "Hello",
			"ST1": *s1,
		},
		"Add": M{
			"A1": "Hello",
			"A2": *s1,
			"I1": 10,
		},
	}
	sel := h.toMgoUpdater(m)
	set := sel["$set"].(map[string]interface{})
	inc := sel["$inc"].(map[string]interface{})
	addToSet := sel["$addToSet"].(map[string]interface{})
	fmt.Println(set["s1"], set["st1"])
	fmt.Println(inc["i1"])
	fmt.Println(addToSet["a1"], addToSet["a2"])
	//Output:Hello ObjectIdHex("513063ef69ca944b1000000a")
	//10
	//Hello ObjectIdHex("513063ef69ca944b1000000a")
}
func ExampleMapToUpdater() {
	ms, err := mgo.Dial("localhost")
	if err != nil {
		panic(err)
	}
	defer ms.Close()
	session := Dial(ms, "rest_test")
	session.DefType(S{})
	session.DefType(SS{})
	rest := session.(*rest)
	m := map[string]interface{}{
		"set": map[string]interface{}{
			"s1":  "Hello",
			"st1": map[string]interface{}{"id": "513063ef69ca944b1000000a"},
		},
		"add": map[string]interface{}{
			"a1": "Hello",
			"a2": map[string]interface{}{"id": "513063ef69ca944b1000000a"},
			"i1": 10,
		},
	}
	sel, err := rest.mapToUpdater(m, baseURL1, reflect.TypeOf(S{}))
	if err != nil {
		panic(err)
	}
	set := sel["Set"].(M)
	inc := sel["Add"].(M)
	fmt.Println(set["S1"], set["ST1"].(SS).id)
	fmt.Println(inc["I1"], inc["A1"], inc["A2"].(SS).id)
	//Output:Hello ObjectIdHex("513063ef69ca944b1000000a")
	//10 Hello ObjectIdHex("513063ef69ca944b1000000a")
}
func ExampleFieldResourcePatch1() {
	ms, err := mgo.Dial("localhost")
	if err != nil {
		panic(err)
	}
	defer ms.Close()
	err = ms.DB("rest_test").C("ss").DropCollection()
	if err != nil {
		panic(err)
	}
	s := Dial(ms, "rest_test")
	s.DefType(SS{})
	s.DefRes("test-ss", FieldResource{
		Type:        "SS",
		Allow:       GET | POST | PATCH,
		PatchFields: []string{"S1"},
	})
	ctx := s.NewContext()
	defer ctx.Close()
	uri, err := ResIdParse("/test-ss")
	if err != nil {
		panic(err)
	}
	r, err := s.R(uri, ctx)
	if err != nil {
		panic(err)
	}
	for i := 0; i < 5; i++ {
		data := SS{S1: fmt.Sprintf("Hello %d", i)}
		_, err := r.Post(&data)
		if err != nil {
			panic(err)
		}
	}
	_, err = r.Patch(M{"Set": M{"S1": "Hello Patch"}})
	if err != nil {
		panic(err)
	}
	resp, err := r.Get()
	if err != nil {
		panic(err)
	}
	iter := resp.(Iter)
	n := iter.Count()
	fmt.Println(n)
	for {
		resp, ok := iter.Next()
		if !ok {
			break
		}
		ss := resp.(*SS)
		fmt.Println(ss.S1)
	}
	var s1set []string
	iter.Extract("S1", &s1set)
	fmt.Println(len(s1set))
	//Output:5
	//Hello Patch
	//Hello Patch
	//Hello Patch
	//Hello Patch
	//Hello Patch
	//1
}
func ExampleFieldResourceDelete2() {
	ms, err := mgo.Dial("localhost")
	if err != nil {
		panic(err)
	}
	defer ms.Close()
	err = ms.DB("rest_test").C("ss").DropCollection()
	if err != nil {
		panic(err)
	}
	s := Dial(ms, "rest_test")
	s.DefType(SS{})
	s.DefRes("test-ss", FieldResource{
		Type:             "SS",
		Allow:            GET | POST | DELETE,
		UpdateWhenDelete: M{"S1": "Deleted"},
	})
	ctx := s.NewContext()
	defer ctx.Close()
	uri, err := ResIdParse("/test-ss")
	if err != nil {
		panic(err)
	}
	r, err := s.R(uri, ctx)
	if err != nil {
		panic(err)
	}
	for i := 0; i < 5; i++ {
		data := SS{S1: fmt.Sprintf("Hello %d", i)}
		_, err := r.Post(&data)
		if err != nil {
			panic(err)
		}
	}
	_, err = r.Delete()
	if err != nil {
		panic(err)
	}
	resp, err := r.Get()
	if err != nil {
		panic(err)
	}
	iter := resp.(Iter)
	n := iter.Count()
	fmt.Println(n)
	for {
		resp, ok := iter.Next()
		if !ok {
			break
		}
		ss := resp.(*SS)
		fmt.Println(ss.S1)
	}
	var s1set []string
	iter.Extract("S1", &s1set)
	fmt.Println(len(s1set))
	//Output:5
	//Deleted
	//Deleted
	//Deleted
	//Deleted
	//Deleted
	//1
}
