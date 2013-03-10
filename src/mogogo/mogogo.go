package mogogo

import (
	"fmt"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"math"
	"net/url"
	"reflect"
	"sort"
	"strings"
	"time"
)

type ErrorCode uint

//Same As HTTP Status
const (
	BadRequest          = 400
	Forbidden           = 403
	Unauthorized        = 401
	NotFound            = 404
	MethodNotAllowed    = 405
	Conflict            = 409
	InternalServerError = 500
)

func (es ErrorCode) String() string {
	var ret string
	switch es {
	case BadRequest:
		ret = "bad request"
	case Unauthorized:
		ret = "unauthorized"
	case Forbidden:
		ret = "forbidden"
	case NotFound:
		ret = "not found"
	case MethodNotAllowed:
		ret = "method not allowed"
	case Conflict:
		ret = "conflict"
	case InternalServerError:
		ret = "internal server error"
	default:
		panic(fmt.Sprintf("invalid errorCode: %d", es))
	}
	return ret
}

type Error struct {
	Code   ErrorCode
	Msg    string
	Err    error
	Fields map[string]error
}

func (re *Error) Error() string {
	var ret string
	if re.Msg != "" {
		ret = re.Msg
	} else {
		ret = re.Code.String()
	}
	if re.Err != nil {
		ret = fmt.Sprintf("%s (%s)", ret, re.Err.Error())
	}
	return ret
}

//被 rest 管理的 struct 必须包含 Base.
type Base struct {
	t      string
	id     bson.ObjectId
	ct     time.Time
	mt     time.Time
	self   interface{}
	r      *rest
	loaded bool
}

var baseType = reflect.TypeOf(Base{})
var urlType = reflect.TypeOf(url.URL{})
var timeType = reflect.TypeOf(time.Time{})

func hasBase(t reflect.Type) bool {
	ft, ok := t.FieldByName("Base")
	if !ok || !ft.Anonymous || ft.Type != baseType {
		return false
	}
	return true
}

func checkHasBase(t reflect.Type) {
	if !hasBase(t) {
		panic(fmt.Sprintf("%s must embed %s", t.Name(), baseType.Name()))
	}
}

func getBase(v reflect.Value) *Base {
	return v.FieldByName("Base").Addr().Interface().(*Base)
}

func (b *Base) Self() *URI {
	return &URI{b.r, []string{typeNameToQueryName(b.t), b.id.Hex()}, nil}
}

func (b *Base) Load() error {
	panic("Not Implements")
}

func (b *Base) R(name string, ctx *Context) Resource {
	panic("Not Implements")
}

//地理位置
type Geo struct {
	Lo float64
	La float64
}

var geoType = reflect.TypeOf(Geo{})

type Method uint

const (
	GET Method = 1 << iota
	PUT
	DELETE
	POST
	PATCH
)

func methodParse(s string) (m Method, ok bool) {
	switch {
	case strings.EqualFold(s, "GET"):
		m = GET
		ok = true
	case strings.EqualFold(s, "PUT"):
		m = PUT
		ok = true
	case strings.EqualFold(s, "DELETE"):
		m = DELETE
		ok = true
	case strings.EqualFold(s, "POST"):
		m = POST
		ok = true
	case strings.EqualFold(s, "PATCH"):
		m = PATCH
		ok = true
	default:
		m = 0
		ok = false
	}
	return
}

func (m Method) String() string {
	var ret string
	switch m {
	case GET:
		ret = "GET"
	case PUT:
		ret = "PUT"
	case DELETE:
		ret = "DELETE"
	case POST:
		ret = "POST"
	case PATCH:
		ret = "PATCH"
	default:
		panic(fmt.Sprintf("invalid method: %#x(%b)", m, m))
	}
	return ret
}

//Field Query
//指定 SortFields 时不可以开启 Pull
//Unique 为 true 时不支持 POST, 为 false 时不支持 PUT
type FieldQuery struct {
	Type       string
	Allow      Method
	Fields     []string
	ContextRef map[string]string
	SortFields []string
	Unique     bool
	Count      bool
	Limit      int
	Pull       bool
}

//Selector Query, 只支持 GET
type SelectorQuery struct {
	ResponseType string
	SelectorFunc func(req *Req, ctx *Context) (selector map[string]interface{}, err error)
	SortFields   []string
	Count        bool
	Limit        uint
}

type Getable interface {
	Get(req *Req, ctx *Context) (result interface{}, err error)
}
type Putable interface {
	Put(req *Req, ctx *Context) (result interface{}, err error)
}
type Deletable interface {
	Delete(req *Req, ctx *Context) (result interface{}, err error)
}
type Postable interface {
	Post(req *Req, ctx *Context) (result interface{}, err error)
}
type Patchable interface {
	Patch(req *Req, ctx *Context) (result interface{}, err error)
}

//Custom Query
type CustomQuery struct {
	RequestType      string
	ResponseType     string
	PathSegmentsType []string
	Handler          interface{}
}

type Context struct {
	r      *rest
	s      *mgo.Session
	Sys    bool
	values map[string]interface{}
	newval bool
}

func (ctx *Context) Get(key string) (val interface{}, ok bool) {
	val, ok = ctx.values[key]
	return
}
func (ctx *Context) Set(key string, val interface{}) {
	ctx.newval = true
	ctx.values[key] = val
}

func (ctx *Context) Close() {
	ctx.s.Close()
	ctx.s = nil
}

func (ctx *Context) coll(typ string) *mgo.Collection {
	if ctx.s == nil {
		panic("context closed")
	}
	return ctx.s.DB(ctx.r.db).C(strings.ToLower(typ))
}

type Req struct {
	*URI
	Method  Method
	Body    interface{}
	RawBody interface{}
}
type Slice interface {
	Prev() *URI
	Next() *URI
	HasCount() bool
	Count() int
	HasLimit()
	Limit() int
	Items() interface{}
}
type Iter interface {
	Count() (n int, err error)
	Err() (err error)
	Next() (result interface{}, ok bool)
	Slice() (slice Slice, err error)
}

type Resource interface {
	NewRequest() interface{}
	Get() (result interface{}, err error)
	Put(request interface{}) (response interface{}, err error)
	Delete() (response interface{}, err error)
	Post(request interface{}) (response interface{}, err error)
	Patch(request interface{}) (response interface{}, err error)
}

type Session interface {
	NewContext() *Context
	DefType(def interface{})
	Def(name string, def interface{})
	Bind(name string, typ string, query string, fields []string)
	Index(typ string, index I)
	R(uri *URI, ctx *Context) (res Resource, err error)
}

type I struct {
	Fields      []string
	Unique      bool
	Sparse      bool
	ExpireAfter time.Duration
}

func Dial(s *mgo.Session, db string) Session {
	return &rest{
		s,
		db,
		make(map[string]reflect.Type),
		make(map[string]*CustomQuery),
		make(map[string]map[string]*bind),
	}
}

type selectorIter struct {
	r          *rest
	typ        reflect.Type
	sortFields []string
	count      bool
	countNum   int
	limit      int
	pull       bool
	uri        *URI
	query      *mgo.Query
	iter       *mgo.Iter
}

func (si *selectorIter) Count() (n int, err error) {
	n, err = si.query.Count()
	if err != nil {
		n, err = 0, &Error{Code: InternalServerError, Err: err}
	}
	return
}
func (si *selectorIter) Err() (err error) {
	if si.iter.Err() == nil {
		err = nil
	} else {
		err = &Error{Code: InternalServerError, Err: si.iter.Err()}
	}
	return
}
func (si *selectorIter) Next() (result interface{}, ok bool) {
	if si.iter == nil {
		si.iter = si.query.Sort(si.sortFields...).Iter()
	}
	b := make(bson.M)
	if si.iter.Next(b) {
		s := reflect.New(si.typ).Interface()
		si.r.bsonToStruct(b, s)
		result, ok = s, true
	} else {
		result, ok = nil, false
	}
	return
}
func (si *selectorIter) Slice() (slice Slice, err error) {
	panic("Not Implement")
}

type rest struct {
	s       *mgo.Session
	db      string
	types   map[string]reflect.Type
	queries map[string]*CustomQuery
	binds   map[string]map[string]*bind
}

func (r *rest) NewContext() *Context {
	return &Context{r: r, s: r.s.Copy(), values: make(map[string]interface{})}
}

type bind struct {
	query  string
	fields []string
}

func getCheckNil(b bson.M, key string) interface{} {
	ret := b[key]
	if ret == nil {
		panic(fmt.Sprintf("key '%s' is nil", key))
	}
	return ret
}
func (r *rest) bsonElemToSlice(v reflect.Value, t reflect.Type) reflect.Value {
	ret := reflect.MakeSlice(t, v.Len(), v.Len())
	for i := 0; i < ret.Len(); i++ {
		ret.Index(i).Set(r.bsonElemToValue(v.Index(i), t.Elem()))
	}
	return ret
}
func (r *rest) bsonElemToStruct(v reflect.Value, t reflect.Type) reflect.Value {
	var ret reflect.Value
	if hasBase(t) {
		s, err := r.newWithObjectId(t, v.Interface().(bson.ObjectId))
		if err != nil {
			panic(err)
		}
		ret = reflect.ValueOf(s).Elem()
	} else if t == urlType {
		s := v.Interface().(string)
		if u, err := url.ParseRequestURI(s); err != nil {
			panic(err)
		} else {
			ret = reflect.ValueOf(*u)
		}
	} else if t == timeType {
		ret = reflect.ValueOf(v.Interface().(time.Time))
	} else if t == geoType {
		lon := v.Index(0).Interface().(float64)
		lat := v.Index(1).Interface().(float64)
		ret = reflect.ValueOf(Geo{La: lat, Lo: lon})
	} else {
		panic(fmt.Sprintf("not support struct type %v", t))
	}
	return ret
}
func (r *rest) bsonElemToValue(v reflect.Value, t reflect.Type) reflect.Value {
	var ret reflect.Value
	switch t.Kind() {
	case reflect.String:
		ret = reflect.New(t).Elem()
		ret.SetString(v.Interface().(string))
	case reflect.Bool:
		ret = reflect.New(t).Elem()
		ret.SetBool(v.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		ret = reflect.New(t).Elem()
		ret.SetInt(v.Int())
	case reflect.Float32, reflect.Float64:
		ret = reflect.New(t).Elem()
		ret.SetFloat(v.Float())
	case reflect.Slice:
		ret = r.bsonElemToSlice(v, t)
	case reflect.Struct:
		ret = r.bsonElemToStruct(v, t)
	default:
		panic(fmt.Sprintf("not support type: '%v'", t))
	}
	return ret
}
func (r *rest) bsonToStruct(b bson.M, s interface{}) {
	v := reflect.ValueOf(s).Elem()
	t := v.Type()
	base := getBase(v)
	base.id = getCheckNil(b, "_id").(bson.ObjectId)
	base.mt = getCheckNil(b, "mt").(time.Time)
	base.ct = getCheckNil(b, "ct").(time.Time)
	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)
		if sf.Anonymous && sf.Type == baseType {
			continue
		}
		fv := v.Field(i)
		elem := b[strings.ToLower(sf.Name)]
		if sf.Type.Kind() == reflect.Ptr {
			if elem != nil {
				fv.Set(r.bsonElemToValue(reflect.ValueOf(elem), sf.Type.Elem()).Addr())
			}
		} else if sf.Type.Kind() == reflect.Slice {
			if elem != nil {
				fv.Set(r.bsonElemToValue(reflect.ValueOf(elem), sf.Type))
			} else {
				fv.Set(reflect.MakeSlice(sf.Type, 0, 0))
			}
		} else {
			if elem == nil {
				panic(fmt.Sprintf("'%v.%s' not nil", v.Type(), sf.Name))
			}
			fv.Set(r.bsonElemToValue(reflect.ValueOf(elem), sf.Type))
		}
	}
	base.loaded = true
	base.r = r
}

func (r *rest) sliceToMapElem(v reflect.Value, t reflect.Type, baseURL *url.URL) interface{} {
	ret := make([]interface{}, v.Len(), v.Len())
	for i := 0; i < len(ret); i++ {
		ret[i] = r.valueToMapElem(v.Index(i), t.Elem(), baseURL)
	}
	return ret
}
func (r *rest) structToMapElem(v reflect.Value, t reflect.Type, baseURL *url.URL) interface{} {
	var ret interface{}
	if hasBase(t) {
		base := getBase(v)
		ret = map[string]interface{}{
			"id":   base.id.Hex(),
			"type": strings.ToLower(base.t),
			"self": base.Self().URLWithBase(baseURL).String(),
		}

	} else if t == urlType {
		url := v.Addr().Interface().(*url.URL)
		if url.Host == "" {
			url.Scheme = baseURL.Scheme
			url.Host = baseURL.Host
		}
		ret = url.String()
	} else if t == timeType {
		tm := v.Interface().(time.Time)
		ret = tm.Format(time.RFC3339)
	} else if t == geoType {
		geo := v.Interface().(Geo)
		ret = map[string]interface{}{"lon": geo.Lo, "lat": geo.La}
	} else {
		panic(fmt.Sprintf("not support struct type %v", t))
	}
	return ret
}
func (r *rest) valueToMapElem(v reflect.Value, t reflect.Type, baseURL *url.URL) interface{} {
	var ret interface{}
	switch t.Kind() {
	case reflect.String:
		fallthrough
	case reflect.Bool:
		fallthrough
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		fallthrough
	case reflect.Float32, reflect.Float64:
		ret = v.Interface()
	case reflect.Slice:
		ret = r.sliceToMapElem(v, t, baseURL)
	case reflect.Struct:
		ret = r.structToMapElem(v, t, baseURL)
	default:
		panic(fmt.Sprintf("not support type: '%v'", t))
	}
	return ret
}
func (r *rest) structToMap(s interface{}, baseURL *url.URL) map[string]interface{} {
	ret := make(map[string]interface{})
	sv := reflect.ValueOf(s).Elem()
	st := sv.Type()
	if hasBase(st) {
		base := getBase(sv)
		if !base.loaded {
			panic("struct not loaded")
		}
		if base.id != "" {
			ret["id"] = base.id.Hex()
			ret["self"] = base.Self().URLWithBase(baseURL).String()
			ret["type"] = base.t
			if base.mt.IsZero() {
				panic("modifiy time not set")
			}
			if base.ct.IsZero() {
				panic("create time not set")
			}
			ret["mt"] = base.mt.Format(time.RFC3339)
			ret["ct"] = base.ct.Format(time.RFC3339)
		}
	}
	for i := 0; i < st.NumField(); i++ {
		sf := st.Field(i)
		key := strings.ToLower(sf.Name)
		if sf.Anonymous && sf.Type == baseType {
			continue
		}
		fv := sv.Field(i)
		if sf.Type.Kind() == reflect.Ptr {
			if !fv.IsNil() {
				ret[key] = r.valueToMapElem(fv.Elem(), sf.Type.Elem(), baseURL)
			}
		} else if sf.Type.Kind() == reflect.Slice {
			if !fv.IsNil() {
				ret[key] = r.valueToMapElem(fv, sf.Type, baseURL)
			} else {
				ret[key] = make([]interface{}, 0)
			}
		} else {
			ret[key] = r.valueToMapElem(fv, sf.Type, baseURL)
		}

	}
	return ret

}
func (r *rest) sliceToBsonElem(v reflect.Value, t reflect.Type) interface{} {
	ret := make([]interface{}, v.Len(), v.Len())
	for i := 0; i < len(ret); i++ {
		ret[i] = r.valueToBsonElem(v.Index(i), t.Elem())
	}
	return ret
}
func (r *rest) structToBsonElem(v reflect.Value, t reflect.Type) interface{} {
	var ret interface{}
	if hasBase(t) {
		ret = getBase(v).id
	} else if t == urlType {
		ret = v.Addr().Interface().(*url.URL).String()
	} else if t == timeType {
		ret = v.Interface()
	} else if t == geoType {
		geo := v.Interface().(Geo)
		ret = []interface{}{geo.Lo, geo.La}
	} else {
		panic(fmt.Sprintf("not support struct type %v", t))
	}
	return ret
}
func (r *rest) valueToBsonElem(v reflect.Value, t reflect.Type) interface{} {
	var ret interface{}
	switch t.Kind() {
	case reflect.String:
		fallthrough
	case reflect.Bool:
		fallthrough
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		fallthrough
	case reflect.Float32, reflect.Float64:
		ret = v.Interface()
	case reflect.Slice:
		ret = r.sliceToBsonElem(v, t)
	case reflect.Struct:
		ret = r.structToBsonElem(v, t)
	default:
		panic(fmt.Sprintf("not support type: '%v'", t))
	}
	return ret
}
func (r *rest) structToBson(s interface{}) bson.M {
	ret := make(bson.M)
	sv := reflect.ValueOf(s).Elem()
	st := sv.Type()
	base := getBase(sv)
	if !base.loaded {
		panic("struct not loaded")
	}
	if base.id != "" {
		ret["_id"] = base.id
		if base.mt.IsZero() {
			panic("modifiy time not set")
		}
		if base.ct.IsZero() {
			panic("create time not set")
		}
		ret["mt"] = base.mt
		ret["ct"] = base.ct
	}
	for i := 0; i < st.NumField(); i++ {
		sf := st.Field(i)
		key := strings.ToLower(sf.Name)
		if sf.Anonymous && sf.Type == baseType {
			continue
		}
		fv := sv.Field(i)
		if sf.Type.Kind() == reflect.Ptr {
			if !fv.IsNil() {
				ret[key] = r.valueToBsonElem(fv.Elem(), sf.Type.Elem())
			}
		} else if sf.Type.Kind() == reflect.Slice {
			if !fv.IsNil() {
				ret[key] = r.valueToBsonElem(fv, sf.Type)
			} else {
				ret[key] = make([]interface{}, 0)
			}
		} else {
			ret[key] = r.valueToBsonElem(fv, sf.Type)
		}

	}
	return ret

}
func (r *rest) mapElemToSlice(v reflect.Value, t reflect.Type, key string, baseURL *url.URL) (reflect.Value, error) {
	ret := reflect.MakeSlice(t, v.Len(), v.Len())
	for i := 0; i < ret.Len(); i++ {
		ki := fmt.Sprintf("%s[%d]", key, i)
		val, err := r.mapElemToValue(v.Index(i), t.Elem(), ki, baseURL)
		if err != nil {
			return reflect.Value{}, err
		}
		if val.Kind() == reflect.Interface {
			val = val.Elem()
		}
		ret.Index(i).Set(val)
	}
	return ret, nil
}
func (r *rest) mapElemToBase(v reflect.Value, t reflect.Type, key string) (reflect.Value, error) {
	var ret reflect.Value
	msg := fmt.Sprintf("field '%s' want {id: objectId}", key)
	obj, ok := v.Interface().(map[string]interface{})
	if !ok {
		return ret, &Error{Code: BadRequest, Msg: msg}
	}
	idi, ok := obj["id"]
	if !ok {
		return ret, &Error{Code: BadRequest, Msg: msg}
	}
	hexId, ok := idi.(string)
	if !ok {
		return ret, &Error{Code: BadRequest, Msg: msg}
	}
	id, err := parseObjectId(hexId)
	if err != nil {
		return ret, &Error{Code: BadRequest, Msg: "field '" + key + "'.id parse error", Err: err}
	}
	s, err := r.newWithObjectId(t, id)
	if err != nil {
		return ret, &Error{Code: BadRequest, Msg: "field '" + key + "'.id", Err: err}
	}
	ret = reflect.ValueOf(s).Elem()
	return ret, nil
}
func (r *rest) mapElemToURL(v reflect.Value, t reflect.Type, key string, baseURL *url.URL) (reflect.Value, error) {
	var ret reflect.Value
	s, ok := v.Interface().(string)
	if !ok {
		return ret, typeError(key, t, v.Type())
	}
	u, err := url.ParseRequestURI(s)
	if err != nil {
		return ret, &Error{Code: BadRequest, Msg: "field '" + key + "' parse error", Err: err}
	}
	if u.Scheme == baseURL.Scheme && u.Host == baseURL.Host {
		u.Scheme = ""
		u.Host = ""
	}
	ret = reflect.ValueOf(*u)
	return ret, nil
}
func (r *rest) mapElemToTime(v reflect.Value, t reflect.Type, key string) (reflect.Value, error) {
	var ret reflect.Value
	s, ok := v.Interface().(string)
	if !ok {
		return ret, typeError(key, t, v.Type())
	}
	tm, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return ret, &Error{Code: BadRequest, Msg: "field '" + key + "'", Err: err}
	}
	ret = reflect.ValueOf(tm)
	return ret, nil
}
func (r *rest) mapElemToGeo(v reflect.Value, t reflect.Type, key string) (reflect.Value, error) {
	var ret reflect.Value
	msg := fmt.Sprintf("field '%s' want {lat:float, lon:float}", key)
	geomap, ok := v.Interface().(map[string]interface{})
	if !ok {
		return ret, &Error{Code: BadRequest, Msg: msg}
	}
	lon, lonOk := geomap["lon"].(float64)
	lat, latOk := geomap["lat"].(float64)
	if !lonOk || !latOk {
		return ret, &Error{Code: BadRequest, Msg: msg}
	}
	ret = reflect.ValueOf(Geo{La: lat, Lo: lon})
	return ret, nil
}
func (r *rest) mapElemToStruct(v reflect.Value, t reflect.Type, key string, baseURL *url.URL) (reflect.Value, error) {
	var ret reflect.Value
	var err error = nil
	if hasBase(t) {
		ret, err = r.mapElemToBase(v, t, key)
	} else if t == urlType {
		ret, err = r.mapElemToURL(v, t, key, baseURL)
	} else if t == timeType {
		ret, err = r.mapElemToTime(v, t, key)
	} else if t == geoType {
		ret, err = r.mapElemToGeo(v, t, key)
	} else {
		panic(fmt.Sprintf("not support struct type %v", t))
	}
	return ret, err
}
func (r *rest) mapElemToInt(v reflect.Value, t reflect.Type, key string) (ret int64, err error) {
	i := v.Interface()
	switch val := i.(type) {
	case int:
		ret, err = int64(val), nil
	case float64:
		n, frac := math.Modf(val)
		if frac != 0.0 {
			ret, err = 0, typeError(key, t, v.Type())
		} else {
			ret, err = int64(n), nil
		}
	case float32:
		n, frac := math.Modf(float64(val))
		if frac != 0.0 {
			ret, err = 0, typeError(key, t, v.Type())
		} else {
			ret, err = int64(n), nil
		}
	case int8:
		ret, err = int64(val), nil
	case int16:
		ret, err = int64(val), nil
	case int32:
		ret, err = int64(val), nil
	case int64:
		ret, err = int64(val), nil
	default:
		ret, err = 0, typeError(key, t, v.Type())
	}
	return
}

func (r *rest) mapElemToFloat(v reflect.Value, t reflect.Type, key string) (ret float64, err error) {

	switch val := v.Interface().(type) {
	case float64:
		ret, err = val, nil
	case float32:
		ret, err = float64(val), nil
	default:
		ret, err = 0, typeError(key, t, v.Type())
	}
	return
}

func typeError(key string, want, but reflect.Type) error {
	msg := fmt.Sprintf("field '%s' want type '%v' but '%v'", key, want, but)
	return &Error{Code: BadRequest, Msg: msg}
}

func (r *rest) mapElemToValue(v reflect.Value, t reflect.Type, key string, baseURL *url.URL) (reflect.Value, error) {
	var ret reflect.Value
	var err error
	switch t.Kind() {
	case reflect.String:
		ret = reflect.New(t).Elem()
		s, ok := v.Interface().(string)
		if !ok {
			return ret, typeError(key, t, v.Type())
		}
		ret.SetString(s)
	case reflect.Bool:
		ret = reflect.New(t).Elem()
		b, ok := v.Interface().(bool)
		if !ok {
			return ret, typeError(key, t, v.Type())
		}
		ret.SetBool(b)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		ret = reflect.New(t).Elem()
		i, err := r.mapElemToInt(v, t, key)
		if err != nil {
			return ret, err
		}
		ret.SetInt(int64(i))
	case reflect.Float32, reflect.Float64:
		ret = reflect.New(t).Elem()
		f, err := r.mapElemToFloat(v, t, key)
		if err != nil {
			return ret, err
		}
		ret.SetFloat(f)
	case reflect.Slice:
		ret, err = r.mapElemToSlice(v, t, key, baseURL)
	case reflect.Struct:
		ret, err = r.mapElemToStruct(v, t, key, baseURL)
	default:
		msg := fmt.Sprintf("field '%s' type '%s' not support'", key, t.Name())
		return ret, &Error{Code: BadRequest, Msg: msg}
	}
	return ret, err
}
func (r *rest) mapToBase(m map[string]interface{}, b *Base) error {
	var err error
	idi, ok := m["id"]
	if !ok {
		return nil
	}
	id, ok := idi.(string)
	if !ok {
		return typeError("id", reflect.TypeOf(""), reflect.TypeOf(idi))
	}
	if b.id, err = parseObjectId(id); err != nil {
		return &Error{Code: BadRequest, Msg: "field 'id' parse error", Err: err}
	}
	cti, ok := m["ct"]
	if !ok {
		return &Error{Code: BadRequest, Msg: "field 'ct' not set"}
	}
	ct, ok := cti.(string)
	if !ok {
		return typeError("ct", reflect.TypeOf(""), reflect.TypeOf(cti))
	}
	if b.ct, err = time.Parse(time.RFC3339, ct); err != nil {
		return &Error{Code: BadRequest, Msg: "field 'ct' parse error", Err: err}
	}
	mti, ok := m["mt"]
	if !ok {
		return &Error{Code: BadRequest, Msg: "field 'mt' not set"}
	}
	mt, ok := mti.(string)
	if !ok {
		return typeError("mt", reflect.TypeOf(""), reflect.TypeOf(mti))
	}
	if b.mt, err = time.Parse(time.RFC3339, mt); err != nil {
		return &Error{Code: BadRequest, Msg: "field 'mt' parse error", Err: err}
	}
	b.r = r
	return nil
}
func (r *rest) mapToStruct(m map[string]interface{}, s interface{}, baseURL *url.URL) error {
	v := reflect.ValueOf(s).Elem()
	t := v.Type()
	var base *Base
	if hasBase(t) {
		base = getBase(v)
		if err := r.mapToBase(m, base); err != nil {
			return err
		}
		base.t = t.Name()
	}
	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)
		if sf.Anonymous && sf.Type == baseType {
			continue
		}
		fv := v.Field(i)
		var v reflect.Value
		var err error = nil
		key := strings.ToLower(sf.Name)
		elem := m[key]
		if sf.Type.Kind() == reflect.Ptr {
			if elem != nil {
				v, err = r.mapElemToValue(reflect.ValueOf(elem), sf.Type.Elem(), key, baseURL)
				if err == nil {
					v = v.Addr()
				}
			}
		} else if sf.Type.Kind() == reflect.Slice {
			if elem != nil {
				v, err = r.mapElemToValue(reflect.ValueOf(elem), sf.Type, key, baseURL)
			} else {
				v = reflect.MakeSlice(sf.Type, 0, 0)
			}
		} else {
			if elem == nil {
				msg := fmt.Sprintf("field '%s' not set", key)
				err = &Error{Code: BadRequest, Msg: msg}
			}
			v, err = r.mapElemToValue(reflect.ValueOf(elem), sf.Type, key, baseURL)
		}
		if err != nil {
			return err
		}
		if v.IsValid() {
			fv.Set(v)
		}
	}
	if base != nil {
		base.loaded = true
	}
	return nil
}
func (r *rest) Bind(name string, typ string, query string, fields []string) {
	r.checkType(typ)
	r.checkQuery(query)
	if name == "" {
		panic("name is empty")
	}
	bt, ok := r.binds[typ]
	if !ok {
		bt = make(map[string]*bind)
		r.binds[typ] = bt
	}
	if _, ok = bt[name]; ok {
		panic(fmt.Sprintf("'%s' already bind", name))
	}
	bt[name] = &bind{query, fields}
}
func (r *rest) registerQuery(name string, cq CustomQuery) {
	checkQueryName(name)
	if _, ok := r.queries[name]; ok {
		panic(fmt.Sprintf("query '%s' already defined", name))
	}
	r.queries[name] = &cq
}
func (r *rest) typeDefined(typ string) bool {
	_, ok := r.types[typ]
	return ok
}
func (r *rest) checkType(typ string) {
	if !r.typeDefined(typ) {
		f := "'%s' not defined"
		panic(fmt.Sprintf(f, typ))
	}
}
func (r *rest) typeByName(name string) reflect.Type {
	r.checkType(name)
	return r.types[name]
}
func (r *rest) checkQuery(query string) {
	if _, ok := r.queries[query]; !ok {
		f := "'%s' not defined"
		panic(fmt.Sprintf(f, query))
	}
}
func (r *rest) DefType(def interface{}) {
	typ := reflect.TypeOf(def)
	if typ.Kind() != reflect.Struct {
		panic("only struct type allowed")
	}
	name := typ.Name()
	if _, ok := r.types[name]; ok {
		panic(fmt.Sprintf("type '%s' already defined", name))
	}
	checkQueryName(strings.ToLower(name))
	r.types[name] = typ
	r.defSelf(name)
}
func (r *rest) defSelf(typ string) {
	r.checkType(typ)
	r.Def(typeNameToQueryName(typ), FieldQuery{
		Type:   typ,
		Fields: []string{"Id"},
		Allow:  GET,
		Unique: true,
	})
}
func (r *rest) Def(name string, def interface{}) {
	switch q := def.(type) {
	case FieldQuery:
		r.defFieldQuery(name, q)
	case SelectorQuery:
		r.defSelectorQuery(name, q)
	case CustomQuery:
		r.defCustomQuery(name, q)
	default:
		panic(fmt.Sprintf("unknown query type: %v", reflect.TypeOf(def)))
	}
}

type fqHandler struct {
	r  *rest
	fq *FieldQuery
}

func newFQHandler(r *rest, fq *FieldQuery) *fqHandler {
	return &fqHandler{r, fq}
}
func setFieldValue(sv reflect.Value, f string, v reflect.Value) error {
	if f != "Id" {
		fv := sv.FieldByName(f)
		if !fv.IsValid() {
			panic(fmt.Sprintf("field '%s' not in '%s'", f, sv.Type().Name()))
		}
		if fv.Kind() == reflect.Ptr {
			if v.Kind() == reflect.Ptr {
				fv.Set(v)
			} else {
				ptr := reflect.New(v.Type())
				ptr.Elem().Set(v)
				fv.Set(ptr)
			}

		} else {
			if v.Kind() == reflect.Ptr {
				fv.Set(v.Elem())
			} else {
				fv.Set(v)
			}
		}
	} else {

		getBase(sv).id = getBase(v.Elem()).id
	}
	return nil
}

func (h *fqHandler) setStructFields(s interface{}, req *Req, ctx *Context) error {
	sv := reflect.ValueOf(s).Elem()
	if h.fq.Fields != nil {
		for i, f := range h.fq.Fields {
			seg, err := req.Segment(i)
			if err != nil {
				return err
			}
			segv := reflect.ValueOf(seg)
			err = setFieldValue(sv, f, segv)
			if err != nil {
				return err
			}
		}
	}
	if h.fq.ContextRef != nil {
		for f, ctxkey := range h.fq.ContextRef {
			c, ok := ctx.Get(ctxkey)
			if !ok {
				msg := fmt.Sprintf("'%s' not in Context", ctxkey)
				return &Error{Code: Unauthorized, Msg: msg}
			}
			err := setFieldValue(sv, f, reflect.ValueOf(c))
			if err != nil {
				return err
			}
		}
	}
	return nil
}
func setBsonValue(b bson.M, f string, v reflect.Value) {
	if f != "Id" {
		if v.Kind() == reflect.Ptr {
			v = v.Elem()
		}
		f = strings.ToLower(f)
		if v.Kind() != reflect.Struct {
			b[f] = v.Interface()
		} else {
			b[f] = getBase(v).id
		}
	} else {
		b["_id"] = getBase(v.Elem()).id
	}
}
func (h *fqHandler) query(req *Req, ctx *Context) (bson.M, error) {
	ret := make(bson.M)
	if h.fq.Fields != nil {
		for i, f := range h.fq.Fields {
			seg, err := req.Segment(i)
			if err != nil {
				return nil, err
			}
			segv := reflect.ValueOf(seg)
			setBsonValue(ret, f, segv)
		}
	}
	if h.fq.ContextRef != nil {
		for f, ctxkey := range h.fq.ContextRef {
			c, ok := ctx.Get(ctxkey)
			if !ok {
				msg := fmt.Sprintf("'%s' not in Context", ctxkey)
				return nil, &Error{Code: Unauthorized, Msg: msg}
			}
			setBsonValue(ret, f, reflect.ValueOf(c))
		}
	}
	return ret, nil
}
func (h *fqHandler) ensureIndex() {
	fields := make([]string, 0)
	if h.fq.Fields != nil {
		fields = append(fields, h.fq.Fields...)
	}
	if h.fq.ContextRef != nil {
		refFields := make([]string, 0)
		for f, _ := range h.fq.ContextRef {
			if _, ok := indexOf(fields, f); !ok {
				refFields = append(refFields, f)
			}
		}
		sort.Strings(refFields)
		fields = append(fields, refFields...)
	}
	if !h.fq.Unique {
		if h.fq.Pull && h.fq.SortFields != nil {
			panic("pull and sort fields")
		}
		if h.fq.SortFields == nil {
			fields = append(fields, "Id")
		} else {
			fields = append(fields, h.fq.SortFields...)
		}
	}
	if len(fields) > 0 {
		idx := I{Fields: fields, Unique: h.fq.Unique}
		h.r.Index(h.fq.Type, idx)
	}
}
func (h *fqHandler) coll(ctx *Context) *mgo.Collection {
	return ctx.coll(h.fq.Type)
}
func (h *fqHandler) Get(req *Req, ctx *Context) (result interface{}, err error) {
	if h.fq.Allow&GET == 0 {
		return nil, &Error{Code: MethodNotAllowed}
	}
	q, err := h.query(req, ctx)
	if err != nil {
		return nil, err
	}
	b := make(bson.M)
	if h.fq.Unique {
		err = h.coll(ctx).Find(q).One(b)
		if err == nil {
			s := h.r.newStruct(h.fq.Type)
			h.r.bsonToStruct(b, s)
			result = s
		} else if err == mgo.ErrNotFound {
			result, err = nil, &Error{Code: NotFound}
		} else {
			result, err = nil, &Error{Code: InternalServerError, Err: err}
		}
	} else {
		sortFields := make([]string, 0)
		if h.fq.SortFields == nil {
			sortFields = append(sortFields, "-Id")
		} else {
			sortFields = append(sortFields, h.fq.SortFields...)
		}
		result, err = &selectorIter{
			r:          h.r,
			typ:        h.r.types[h.fq.Type],
			sortFields: h.r.fieldsToKeys(h.r.types[h.fq.Type], sortFields),
			count:      h.fq.Count,
			limit:      h.fq.Limit,
			pull:       h.fq.Pull,
			uri:        req.URI,
			query:      ctx.coll(h.fq.Type).Find(q),
		}, err

	}
	return
}
func (h *fqHandler) Put(req *Req, ctx *Context) (result interface{}, err error) {
	if h.fq.Allow&PUT == 0 {
		return nil, &Error{Code: MethodNotAllowed}
	}
	q, err := h.query(req, ctx)
	if err != nil {
		return nil, err
	}
	body := req.Body
	err = h.setStructFields(body, req, ctx)
	old := make(bson.M)
	err = h.coll(ctx).Find(q).One(old)
	if err == mgo.ErrNotFound {
		base := getBase(reflect.ValueOf(body).Elem())
		base.id = bson.NewObjectId()
		base.mt = bson.Now()
		base.ct = base.mt
		base.loaded = true
		base.r = h.r
		base.t = h.fq.Type
		b := h.r.structToBson(body)
		err = h.coll(ctx).Insert(b)
		if err != nil {
			lasterr := err.(*mgo.LastError)
			if lasterr.Code == 11000 {
				return nil, &Error{Code: Conflict}
			} else {
				return nil, &Error{Code: InternalServerError, Err: err}
			}
		}
	} else if err == nil {
		base := getBase(reflect.ValueOf(body).Elem())
		base.id = old["_id"].(bson.ObjectId)
		base.mt = bson.Now()
		base.ct = old["ct"].(time.Time)
		base.loaded = true
		base.r = h.r
		base.t = h.fq.Type
		b := h.r.structToBson(body)
		_, err = h.coll(ctx).UpsertId(base.id, b)
		if err != nil {
			lasterr := err.(*mgo.LastError)
			if lasterr.Code == 11000 {
				return nil, &Error{Code: Conflict}
			} else {
				return nil, &Error{Code: InternalServerError, Err: err}
			}
		}

	} else {
		return nil, &Error{Code: InternalServerError, Err: err}
	}
	return body, nil
}
func (h *fqHandler) Delete(req *Req, ctx *Context) (result interface{}, err error) {
	if h.fq.Allow&DELETE == 0 {
		return nil, &Error{Code: MethodNotAllowed}
	}
	q, err := h.query(req, ctx)
	if err != nil {
		return nil, err
	}
	_, err = h.coll(ctx).RemoveAll(q)
	if err != nil {
		err = &Error{Code: InternalServerError, Err: err}
	}
	return nil, err
}
func (h *fqHandler) Post(req *Req, ctx *Context) (result interface{}, err error) {
	if h.fq.Allow&POST == 0 {
		return nil, &Error{Code: MethodNotAllowed}
	}
	body := req.Body
	err = h.setStructFields(body, req, ctx)
	if err != nil {
		return nil, err
	}
	base := getBase(reflect.ValueOf(body).Elem())
	base.id = bson.NewObjectId()
	base.mt = bson.Now()
	base.ct = base.mt
	base.loaded = true
	base.r = h.r
	base.t = h.fq.Type
	b := h.r.structToBson(body)
	err = h.coll(ctx).Insert(b)
	if err != nil {
		lasterr := err.(*mgo.LastError)
		if lasterr.Code == 11000 {
			return nil, &Error{Code: Conflict}
		} else {
			return nil, &Error{Code: InternalServerError, Err: err}
		}
	}
	return body, nil
}
func (h *fqHandler) Patch(req *Req, ctx *Context) (result interface{}, err error) {
	panic("Not Implement")
}

func (r *rest) fieldsToPathSegmentsType(t reflect.Type, fields []string) []string {
	ret := make([]string, 0)
	if fields == nil {
		return ret
	}
	for _, field := range fields {
		if field == "Id" {
			ret = append(ret, t.Name())
			continue
		}
		if field == "CT" || field == "MT" {
			panic(fmt.Sprintf("segment type not support type '%s'", "time.Time"))
		}
		sf, ok := t.FieldByName(field)
		if !ok {
			panic(fmt.Sprintf("field '%s' not in '%s'", field, t.Name()))
		}
		ft := sf.Type
		if ft.Kind() == reflect.Ptr {
			ft = ft.Elem()
		}
		switch ft.Kind() {
		case reflect.Int:
			ret = append(ret, "int")
		case reflect.String:
			ret = append(ret, "string")
		case reflect.Bool:
			ret = append(ret, "bool")
		case reflect.Struct:
			r.checkType(ft.Name())
			r.checkHasBase(ft.Name())
			ret = append(ret, ft.Name())
		default:
			panic(fmt.Sprintf("segment type not support type '%v'", ft))
		}

	}
	return ret
}
func checkFieldQuery(fq *FieldQuery) {
	if fq.Allow&PUT != 0 && !fq.Unique {
		panic("PUT only support unique field query")
	}
}
func (r *rest) defFieldQuery(name string, fq FieldQuery) {
	r.checkType(fq.Type)
	checkFieldQuery(&fq)
	h := newFQHandler(r, &fq)
	h.ensureIndex()
	segtype := r.fieldsToPathSegmentsType(r.types[fq.Type], fq.Fields)
	cq := CustomQuery{fq.Type, fq.Type, segtype, h}
	r.defCustomQuery(name, cq)
}
func (r *rest) defSelectorQuery(name string, sq SelectorQuery) {
	r.checkType(sq.ResponseType)
	panic("Not Implement")
}
func (r *rest) checkPathSegmentsType(segtype []string) {
	for _, e := range segtype {
		if r.typeDefined(e) {
			continue
		}
		switch e {
		case "int", "string", "bool":
			continue
		}
		panic(fmt.Sprintf("type '%s' not support", e))
	}
}
func (r *rest) defCustomQuery(name string, cq CustomQuery) {
	r.checkType(cq.RequestType)
	r.checkType(cq.ResponseType)
	r.checkPathSegmentsType(cq.PathSegmentsType)
	if cq.Handler == nil {
		panic("Handler can't be nil")
	}
	r.registerQuery(name, cq)
}
func (r *rest) fieldsToKeys(typ reflect.Type, fields []string) []string {
	inidx := make(map[string]bool)
	ret := make([]string, 0)
	for _, field := range fields {
		f := field
		p := ""
		if strings.HasPrefix(f, "-") || strings.HasPrefix(f, "@") {
			p = f[0:1]
			f = f[1:]
		}
		if inidx[f] {
			panic(fmt.Sprintf("duplicate field '%s'", f))
		}
		inidx[f] = true
		_, hf := typ.FieldByName(f)
		if f == "Id" {
			ret = append(ret, p+"_id")
		} else if hf || f == "MT" || f == "CT" {
			ret = append(ret, p+strings.ToLower(f))
		} else {
			panic(fmt.Sprintf("field '%s' not set in '%s'", f, typ))
		}
	}
	return ret
}
func (r *rest) checkHasBase(typ string) {
	checkHasBase(r.types[typ])
}
func (r *rest) Index(typ string, index I) {
	r.checkType(typ)
	r.checkHasBase(typ)
	c := r.s.DB(r.db).C(strings.ToLower(typ))
	mgoidx := mgo.Index{
		Key:         r.fieldsToKeys(r.types[typ], index.Fields),
		Unique:      index.Unique,
		Sparse:      index.Sparse,
		ExpireAfter: index.ExpireAfter,
	}
	err := c.EnsureIndex(mgoidx)
	if err != nil {
		panic(err)
	}
}
func (r *rest) newWithObjectId(typ reflect.Type, id bson.ObjectId) (val interface{}, err error) {
	v := reflect.New(typ)
	b := getBase(v.Elem())
	b.id = id
	b.t = typ.Name()
	b.r = r
	b.self = v.Interface()
	return b.self, nil
}
func (r *rest) newStruct(typ string) interface{} {
	v := reflect.New(r.types[typ])
	b := getBase(v.Elem())
	b.t = typ
	b.r = r
	b.self = v.Interface()
	return b.self
}
func (r *rest) newWithId(typ string, hex string) (val interface{}, err error) {
	id, err := parseObjectId(hex)
	if err != nil {
		return nil, fmt.Errorf("parse object id error: %v", err)
	}
	return r.newWithObjectId(r.types[typ], id)
}

type resource struct {
	cq  *CustomQuery
	uri *URI
	ctx *Context
	r   *rest
}

func (res *resource) requestToBody(req interface{}) (body interface{}, err error) {
	defRequestType := res.r.types[res.cq.RequestType]
	requestType := reflect.TypeOf(req)
	if requestType.Kind() == reflect.Ptr && requestType.Elem() == defRequestType {
		body, err = req, nil
	} else {
		panic(fmt.Sprintf("request type want: %v, got %v", reflect.PtrTo(defRequestType), requestType))
	}
	return
}
func (res *resource) checkResponse(val interface{}, err error) {
	responseType := res.r.types[res.cq.ResponseType]
	if val == nil {
		return
	}
	resultType := reflect.TypeOf(val)
	if resultType.Kind() == reflect.Ptr && resultType.Elem() == responseType {
		return
	}
	if _, ok := val.(Iter); ok {
		return
	}
	panic(fmt.Sprintf("not support response type: %v", resultType))
}
func (res *resource) Get() (response interface{}, err error) {
	getable, ok := res.cq.Handler.(Getable)
	if !ok {
		return nil, &Error{Code: MethodNotAllowed}
	}
	req := &Req{URI: res.uri, Method: GET}
	response, err = getable.Get(req, res.ctx)
	res.checkResponse(response, err)
	return
}

func (res *resource) Put(request interface{}) (response interface{}, err error) {
	putable, ok := res.cq.Handler.(Putable)
	if !ok {
		return nil, &Error{Code: MethodNotAllowed}
	}
	body, err := res.requestToBody(request)
	if err != nil {
		return nil, err
	}
	req := &Req{URI: res.uri, Method: GET, Body: body, RawBody: request}
	response, err = putable.Put(req, res.ctx)
	res.checkResponse(response, err)
	return
}

func (res *resource) Delete() (response interface{}, err error) {
	deletable, ok := res.cq.Handler.(Deletable)
	if !ok {
		return nil, &Error{Code: MethodNotAllowed}
	}
	req := &Req{URI: res.uri, Method: GET}
	response, err = deletable.Delete(req, res.ctx)
	res.checkResponse(response, err)
	return
}

func (res *resource) Post(request interface{}) (response interface{}, err error) {
	postable, ok := res.cq.Handler.(Postable)
	if !ok {
		return nil, &Error{Code: MethodNotAllowed}
	}
	body, err := res.requestToBody(request)
	if err != nil {
		return nil, err
	}
	req := &Req{URI: res.uri, Method: GET, Body: body, RawBody: request}
	response, err = postable.Post(req, res.ctx)
	res.checkResponse(response, err)
	return
}

func (res *resource) Patch(request interface{}) (response interface{}, err error) {
	patchable, ok := res.cq.Handler.(Patchable)
	if !ok {
		return nil, &Error{Code: MethodNotAllowed}
	}

	req := &Req{URI: res.uri, Method: GET, Body: nil, RawBody: request}
	response, err = patchable.Patch(req, res.ctx)
	res.checkResponse(response, err)
	return
}
func (res *resource) NewRequest() interface{} {
	return reflect.New(res.r.types[res.cq.RequestType]).Interface()
}
func (r *rest) queryRes(cq *CustomQuery, uri *URI, ctx *Context) (res Resource, err error) {
	return &resource{cq, uri, ctx, r}, nil
}
func (r *rest) R(uri *URI, ctx *Context) (res Resource, err error) {
	uri.r = r
	name := uri.path[0]
	if qry, ok := r.queries[name]; ok {
		if isSysQueryName(name) && !ctx.Sys {
			return nil, &Error{Code: Forbidden, Msg: "private url"}
		}
		return r.queryRes(qry, uri, ctx)
	}
	msg := fmt.Sprintf("'%s' not found", uri.String())
	return nil, &Error{Code: NotFound, Msg: msg}
}
