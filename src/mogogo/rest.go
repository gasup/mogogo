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
	Fields map[string]string
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
	r      *rest
	loaded bool
}

var baseType reflect.Type = reflect.TypeOf(Base{})

func hasBase(t reflect.Type) bool {
	ft, ok := t.FieldByName("Base")
	if !ok || !ft.Anonymous || ft.Type != baseType {
		return false
	}
	return true
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
	Type  string
	Allow Method
	//可以通过 ":" 引用 Context. 比如 "user:currentUser"
	Fields     []string
	SortFields []string
	Unique     bool
	Count      bool
	Limit      uint
	Pull       bool
}

//Selector Query, 只支持 GET
type SelectorQuery struct {
	ResultType   string
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
	BodyType   string
	ResultType string
	ElemType   []string
	Handler    interface{}
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
	Put(body interface{}) (result interface{}, err error)
	Delete() (result interface{}, err error)
	Post(body interface{}) (result interface{}, err error)
	Patch(body interface{}) (result interface{}, err error)
}

type Session interface {
	DefType(def interface{})
	Def(name string, def interface{})
	Bind(name string, typ string, query string, fields []string, ctxref map[string]string)
	Index(typ string, index I)
	R(uri *URI, ctx *Context) (res Resource, err error)
}

type I struct {
	Key         []string
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
	ctxref map[string]string
}
type stage int

const (
	req stage = iota
	store
)

func (r *rest) mapToStruct(m map[string]interface{}, typ string, base *url.URL, stg stage) (s interface{}, err error) {
	panic("Not Implement")
}
func (r *rest) structToMap(s interface{}, stg stage) (m map[string]interface{}, err error) {
	panic("Not Implement")
}
func (r *rest) Bind(name string, typ string, query string, fields []string, ctxref map[string]string) {
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
		panic(fmt.Sprintln("'%s' already bind", name))
	}
	bt[name] = &bind{query, fields, ctxref}
}
func (r *rest) registerQuery(name string, cq CustomQuery) {
	checkQueryName(name)
	if _, ok := r.queries[name]; ok {
		panic(fmt.Sprintln("query '%s' already defined", name))
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
		panic(fmt.Sprintln(f, typ))
	}
}
func (r *rest) checkQuery(query string) {
	if _, ok := r.queries[query]; !ok {
		f := "'%s' not defined"
		panic(fmt.Sprintln(f, query))
	}
}
func (r *rest) DefType(def interface{}) {
	typ := reflect.TypeOf(def)
	if typ.Kind() != reflect.Struct {
		panic("only struct type allowed")
	}
	name := typ.Name()
	if _, ok := r.types[name]; ok {
		panic(fmt.Sprintln("type '%s' already defined", name))
	}
	checkQueryName(strings.ToLower(name))
	r.types[name] = typ
	r.defSelf(name)
}
func (r *rest) defSelf(typ string) {
	r.checkType(typ)
	r.Def(typeNameToQueryName(typ), FieldQuery{
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
		panic(fmt.Sprintln("unknown query type: %v", reflect.TypeOf(def)))
	}
}

func (r *rest) defFieldQuery(name string, fq FieldQuery) {
	r.checkType(fq.Type)
	panic("Not Implement")
}
func (r *rest) defSelectorQuery(name string, sq SelectorQuery) {
	r.checkType(sq.ResultType)
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
	r.checkType(cq.BodyType)
	r.checkType(cq.ResultType)
	r.checkElemType(cq.ElemType)
	if cq.Handler == nil {
		panic("Handler can't be nil")
	}
	r.registerQuery(name, cq)
}
func (r *rest) Index(typ string, index I) {
	r.checkType(typ)
	c := r.s.DB(r.db).C(strings.ToLower(typ))
	mgoidx := mgo.Index{
		Key:         index.Key,
		Unique:      index.Unique,
		Sparse:      index.Sparse,
		ExpireAfter: index.ExpireAfter,
	}
	err := c.EnsureIndex(mgoidx)
	if err != nil {
		panic(err)
	}
}

func (r *rest) newWithId(typ string, id string) (val interface{}, err error) {
	v := reflect.New(r.types[typ])
	b := v.FieldByName("Base").Addr().Interface().(*Base)
	d, err := hex.DecodeString(id)
	if err != nil || len(d) != 12 {
		return nil, fmt.Errorf("id format error: %s", id)
	}
	b.id = bson.ObjectId(d)
	b.t = typ
	b.r = r
	return v.Interface(), nil
}

type resource struct {
	cq  *CustomQuery
	uri *URI
	ctx *Context
	r   *rest
}

func (res *resource) checkResult(val interface{}, err error) bool {
	defResultType := res.r.types[res.cq.ResultType]
	resultType := reflect.TypeOf(val)
	if resultType.Kind() == reflect.Ptr && resultType.Elem() == defResultType {
		return true
	}
	if _, ok := val.(Iter); ok {
		return true
	}
	if val == nil && err != nil {
		return true
	}
	return false
}
func (res *resource) Get() (result interface{}, err error) {
	getable, ok := res.cq.Handler.(Getable)
	if !ok {
		return nil, &Error{Code: MethodNotAllowed}
	}
	req := &Req{URI: res.uri, Method: GET}
	result, err = getable.Get(req, res.ctx)
	if res.checkResult(result, err) {
		return
	}
	panic("Not Implement")

}

func (res *resource) Put(body interface{}) (result interface{}, err error) {
	putable, ok := res.cq.Handler.(Putable)
	if !ok {
		return nil, &Error{Code: MethodNotAllowed}
	}
	req := &Req{URI: res.uri, Method: GET, Body:nil, RawBody:body}
	result, err = putable.Put(req, res.ctx)
	if res.checkResult(result, err) {
		return
	}
	panic("Not Implement")
}

func (res *resource) Delete() (result interface{}, err error) {
	deletable, ok := res.cq.Handler.(Deletable)
	if !ok {
		return nil, &Error{Code: MethodNotAllowed}
	}
	req := &Req{URI: res.uri, Method: GET}
	result, err = deletable.Delete(req, res.ctx)
	if res.checkResult(result, err) {
		return
	}
	panic("Not Implement")

}

func (res *resource) Post(body interface{}) (result interface{}, err error) {
	postable, ok := res.cq.Handler.(Postable)
	if !ok {
		return nil, &Error{Code: MethodNotAllowed}
	}
	req := &Req{URI: res.uri, Method: GET, Body:nil, RawBody:body}
	result, err = postable.Post(req, res.ctx)
	if res.checkResult(result, err) {
		return
	}
	panic("Not Implement")

}

func (res *resource) Patch(body interface{}) (result interface{}, err error) {
	patchable, ok := res.cq.Handler.(Patchable)
	if !ok {
		return nil, &Error{Code: MethodNotAllowed}
	}
	req := &Req{URI: res.uri, Method: GET, Body:nil, RawBody:body}
	result, err = patchable.Patch(req, res.ctx)
	if res.checkResult(result, err) {
		return
	}
	panic("Not Implement")

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
