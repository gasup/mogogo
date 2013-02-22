package mogogo

import (
	"fmt"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"testing"
	"time"
	"net/url"
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
	F1  float32
	F2  float64
	A1  []string
	A2  []SS
	A3  []string
	G1  Geo
	T1  time.Time
	ST1 SS
	U1  url.URL
	U2  url.URL
}

var t1 = time.Now()
var st1 = bson.NewObjectId()
var b = bson.M{
	"_id": bson.NewObjectId(),
	"ct":  time.Now(),
	"mt":  time.Now(),
	"s1":  "Hello World",
	"s2":  "Liu Dian",
	"s3":  "Pointer",
	"b1":  true,
	"i1":  1,
	"i2":  2,
	"i3":  3,
	"f1":  3.0,
	"f2":  6.0,
	"a1":  []interface{}{"a", "b", "c"},
	"a2":  []interface{}{bson.NewObjectId(), bson.NewObjectId(), bson.NewObjectId()},
	"g1":  []float64{1.0, 2.0},
	"t1":  t1,
	"st1": st1,
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
	rest.bsonToStruct(b, &s)
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
	if s.T1 != t1 {
		t.Error("Time")
	}
	if s.ST1.id != st1 {
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
	rest.bsonToStruct(b, &s)
	bb := rest.structToBson(&s)
	if bb["s1"].(string) != "Hello World" {
		t.Error("structToBson")
	}
}
