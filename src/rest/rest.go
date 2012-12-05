package rest

import (
	"fmt"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
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
		ret = "Bad Request"
	case Unauthorized:
		ret = "Unauthorized"
	case Forbidden:
		ret = "Forbidden"
	case NotFound:
		ret = "Not Found"
	case MethodNotAllowed:
		ret = "Method Not Allowed"
	case Conflict:
		ret = "Conflict"
	default:
		panic(fmt.Sprintf("invalid errorCode: %d", es))
	}
	return ret
}

type RESTError struct {
	Code   ErrorCode
	Msg    string
	Err    error
	Fields map[string]string
}

func (re *RESTError) Error() string {
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
	rest   REST
	loaded bool
}

var baseType reflect.Type = reflect.TypeOf(Base{})

func (b *Base) Self() string {
	return "/" + typeNameToQueryName(b.t) + "/" + b.id.Hex()
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
	BodyType     string
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
	Delete() (err error)
	Post(body interface{}) (result interface{}, err error)
	Patch(body interface{}) (result interface{}, err error)
}

type REST interface {
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

func NewREST(s *mgo.Session, db string) REST {
	return &rest{
		s,
		db,
		make(map[string]reflect.Type),
		make(map[string]interface{}),
		make(map[string]map[string]bind),
	}
}

type rest struct {
	s       *mgo.Session
	db      string
	types   map[string]reflect.Type
	queries map[string]interface{}
	binds   map[string]map[string]bind
}

type bind struct {
	query  string
	fields []string
	ctxref map[string]string
}

func (r *rest) Bind(name string, typ string, query string, fields []string, ctxref map[string]string) {
	r.checkType(typ)
	r.checkQuery(query)
	if name == "" {
		panic("name is empty")
	}
	bt, ok := r.binds[typ]
	if !ok {
		bt = make(map[string]bind)
		r.binds[typ] = bt
	}
	if _, ok = bt[name]; ok {
		panic(fmt.Sprintln("'%s' already bind", name))
	}

}
func (r *rest) registerQuery(name string, q interface{}) {
	checkQueryName(name)
	if _, ok := r.queries[name]; ok {
		panic(fmt.Sprintln("query '%s' already defined", name))
	}
	switch q.(type) {
	case FieldQuery, SelectorQuery, CustomQuery:
		r.queries[name] = q
	default:
		panic(fmt.Sprintln("unknown query type: %v", reflect.TypeOf(q)))
	}
}
func (r *rest) checkType(typ string) {
	if _, ok := r.types[typ]; !ok {
		f := "'%s' not defined"
		panic(fmt.Sprintln(f, typ))
	}
	checkQueryName(strings.ToLower(typ))
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
	r.registerQuery(name, fq)
}
func (r *rest) defSelectorQuery(name string, sq SelectorQuery) {
	r.checkType(sq.BodyType)
	r.checkType(sq.ResultType)
	r.registerQuery(name, sq)
}
func (r *rest) defCustomQuery(name string, cq CustomQuery) {
	r.checkType(cq.BodyType)
	r.checkType(cq.ResultType)
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
func (r *rest) typeRes(t reflect.Type, uri *URI, ctx *Context) (res Resource, err error) {
	panic("Not Implement")
}

type resource struct {
}

func (res *resource) Get() Iter {
	panic("Not Implement")

}
func (res *resource) GetSlice() (slice Slice, err error) {
	panic("Not Implement")

}

func (res *resource) GetOne() (result interface{}, err error) {
	panic("Not Implement")

}

func (res *resource) Put(body interface{}) (result interface{}, err error) {
	panic("Not Implement")

}

func (res *resource) Delete() (err error) {
	panic("Not Implement")

}

func (res *resource) Post(body interface{}) (result interface{}, err error) {
	panic("Not Implement")

}

func (res *resource) Patch(body interface{}) (result interface{}, err error) {
	panic("Not Implement")

}

func (r *rest) queryRes(query interface{}, uri *URI, ctx *Context) (res Resource, err error) {
	panic("Not Implement")
}
func (r *rest) R(uri *URI, ctx *Context) (res Resource, err error) {
	name := uri.Path[0]
	if qry, ok := r.queries[name]; ok {
		if isSysQueryName(name) && !ctx.Sys {
			return nil, &RESTError{Code: Forbidden, Msg: uri.String()}
		}
		return r.queryRes(qry, uri, ctx)
	}
	return nil, &RESTError{Code: NotFound, Msg: uri.String()}
}
