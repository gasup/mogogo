package mogogo

import (
	"encoding/hex"
	"fmt"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"net/url"
	"reflect"
	"strings"
	"time"
)

type ErrorCode uint

//Same As HTTP Status
const (
	BadRequest       = 400
	Forbidden        = 403
	Unauthorized     = 401
	NotFound         = 404
	MethodNotAllowed = 405
	Conflict         = 409
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
	Limit      uint
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
	RequestType  string
	ResponseType string
	ElemType     []string
	Handler      interface{}
}

type Context struct {
	Sys    bool
	values map[string]interface{}
	newval bool
}

func (ctx *Context) Get(key string) (val interface{}, ok bool) {
	panic("Not Implements")
}
func (ctx *Context) Set(key string, val interface{}) {
	panic("Not Implements")
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
	CanCursor() bool
	Cursor() *URI
	Next() (result interface{}, err error)
	Slice() (slice Slice, err error)
}
type Resource interface {
	Get() (result interface{}, err error)
	Put(request interface{}) (response interface{}, err error)
	Delete() (response interface{}, err error)
	Post(request interface{}) (response interface{}, err error)
	Patch(request interface{}) (response interface{}, err error)
}

type Session interface {
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

type rest struct {
	s       *mgo.Session
	db      string
	types   map[string]reflect.Type
	queries map[string]*CustomQuery
	binds   map[string]map[string]*bind
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
		} else if  sf.Type.Kind() == reflect.Slice {
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
func  (r *rest) valueToBsonElem(v reflect.Value, t reflect.Type) interface{} {
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
			panic("modifiy time is zero")
		}
		if base.ct.IsZero() {
			panic("create time is zero")
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
func (h *fqHandler) ensureIndex() {
	fields := h.fq.Fields
	if h.fq.ContextRef != nil {
		for f, _ := range h.fq.ContextRef {
			if _, ok := indexOf(fields, f); !ok {
				fields = append(fields, f)
			}
		}
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
	idx := I{Fields: fields, Unique: h.fq.Unique}
	h.r.Index(h.fq.Type, idx)
}
func (h *fqHandler) Get(req *Req, ctx *Context) (result interface{}, err error) {
	panic("Not Implement")
}
func (h *fqHandler) Put(req *Req, ctx *Context) (result interface{}, err error) {
	panic("Not Implement")
}
func (h *fqHandler) Delete(req *Req, ctx *Context) (result interface{}, err error) {
	panic("Not Implement")
}
func (h *fqHandler) Post(req *Req, ctx *Context) (result interface{}, err error) {
	panic("Not Implement")
}
func (h *fqHandler) Patch(req *Req, ctx *Context) (result interface{}, err error) {
	panic("Not Implement")
}

func (r *rest) fieldsToElemType(t reflect.Type, fields []string) []string {
	ret := make([]string, 0)
	for _, field := range fields {
		if field == "Id" {
			ret = append(ret, t.Name())
			continue
		}
		if field == "CT" || field == "MT" {
			panic(fmt.Sprintf("elem type not support type '%s'", "time.Time"))
		}
		sf, ok := t.FieldByName(field)
		if !ok {
			panic(fmt.Sprintf("field '%s' not found in '%s'", field, t.Name()))
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
			panic(fmt.Sprintf("elem type not support type '%v'", ft))
		}

	}
	return ret
}
func (r *rest) defFieldQuery(name string, fq FieldQuery) {
	r.checkType(fq.Type)
	h := newFQHandler(r, &fq)
	elemtype := r.fieldsToElemType(r.types[fq.Type], fq.Fields)
	cq := CustomQuery{fq.Type, fq.Type, elemtype, h}
	r.defCustomQuery(name, cq)
}
func (r *rest) defSelectorQuery(name string, sq SelectorQuery) {
	r.checkType(sq.ResponseType)
	panic("Not Implement")
}
func (r *rest) checkElemType(elemtype []string) {
	for _, e := range elemtype {
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
	r.checkElemType(cq.ElemType)
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
			panic(fmt.Sprintf("field '%s' not found in '%s'", f, typ))
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
func (r *rest) newWithId(typ string, id string) (val interface{}, err error) {
	d, err := hex.DecodeString(id)
	if err != nil || len(d) != 12 {
		return nil, fmt.Errorf("id format error: %s", id)
	}
	return r.newWithObjectId(r.types[typ], bson.ObjectId(d))
}
func mapToType(m map[string]interface{}, t reflect.Type) (val interface{}, err error) {
	panic("Not Implement")
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
	} else if m, ok := req.(map[string]interface{}); ok {
		body, err = mapToType(m, defRequestType)
	} else {
		panic(fmt.Sprintf("can't support request type: %v", requestType))
	}
	return
}
func (res *resource) checkResponse(val interface{}, err error) {
	responseType := res.r.types[res.cq.ResponseType]
	resultType := reflect.TypeOf(val)
	if resultType.Kind() == reflect.Ptr && resultType.Elem() == responseType {
		return
	}
	if _, ok := val.(Iter); ok {
		return
	}
	if val == nil && err != nil {
		return
	}
	panic(fmt.Sprintf("can't support response type: %v", resultType))
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

func (r *rest) queryRes(cq *CustomQuery, uri *URI, ctx *Context) (res Resource, err error) {
	return &resource{cq, uri, ctx, r}, nil
}
func (r *rest) R(uri *URI, ctx *Context) (res Resource, err error) {
	name := uri.path[0]
	if qry, ok := r.queries[name]; ok {
		if isSysQueryName(name) && !ctx.Sys {
			return nil, &Error{Code: Forbidden, Msg: "private url"}
		}
		return r.queryRes(qry, uri, ctx)
	}
	return nil, &Error{Code: NotFound, Msg: uri.String()}
}
