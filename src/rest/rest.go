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
		panic(fmt.Sprintf("Invalid ErrorCode: %d", es))
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

func (b *Base) Self() string {
	return "/" + b.t + "/" + b.id.Hex()
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
		panic(fmt.Sprintf("Invalid Method: %#x(%b)", m, m))
	}
	return ret
}

//Field Query
//指定 SortFields 时不可以开启 Pull
//Unique 为 true 时不支持 POST, 为 false 时不支持 PUT
type FQ struct {
	Type  interface{}
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
type SQ struct {
	BodyType     interface{}
	ResultType   interface{}
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
type CQ struct {
	BodyType   interface{}
	ResultType interface{}
	Handler    interface{}
}

type Context struct {
	Sys    bool
	val    interface{}
	newval bool
}

func (ctx *Context) Get() (val interface{}, ok bool) {
	panic("Not Implements")
}
func (ctx *Context) Set(newval interface{}) {
}

type Req struct {
	*URI
	Method  Method
	Body    interface{}
	RawBody map[string]interface{}
}
type Iter interface {
	CanCursor() bool
	Cursor() *URI
	Next() (result interface{}, err error)
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
type Resource interface {
	Get() Iter
	GetSlice() (slice Slice, err error)
	GetOne() (result interface{}, err error)
	Put(body interface{}) (result interface{}, err error)
	Delete() (err error)
	Post(body interface{}) (result interface{}, err error)
	Patch(body interface{}) (result interface{}, err error)
}

type REST interface {
	FieldQuery(name string, fq FQ)
	SelectorQuery(name string, sq SQ)
	CustomQuery(name string, cq CQ)
	Index(typ interface{}, index Index)
	R(uri *URI, ctx *Context) (res Resource, err error)
}

type Index struct {
	Key         []string
	Unique      bool
	Sparse      bool
	ExpireAfter time.Duration
}

func NewREST(s *mgo.Session, db string) REST {
	return &rest{s, db, make(map[string]reflect.Type), make(map[string]interface{})}
}

type rest struct {
	s       *mgo.Session
	db      string
	types   map[string]reflect.Type
	queries map[string]interface{}
}

func (r *rest) registerType(t interface{}) {
	typ := reflect.TypeOf(t)
	if typ.Kind() != reflect.Struct {
		panic("only struct type allowed")
	}
	name := strings.ToLower(typ.Name())
	if told, ok := r.types[name]; ok {
		if typ != told {
			f := "type of '%s' must be %v"
			panic(fmt.Sprintln(f, name, told))
		}
	} else {
		r.types[name] = typ
	}
}

func (r *rest) typeName(typ interface{}) string {
	t := reflect.TypeOf(typ)
	name := strings.ToLower(t.Name())
	if _, ok := r.types[name]; !ok {
		panic(fmt.Sprintf("type '%v' not register", t))
	}
	return name
}
func (r *rest) registerQuery(name string, q interface{}) {
	checkQueryName(name)
	if _, ok := r.queries[name]; ok {
		panic(fmt.Sprintln("query '%s' already defined", name))
	}
	switch t := q.(type) {
	case FQ, SQ, CQ:
		r.queries[name] = q
	default:
		panic(fmt.Sprintln("unknown query type: %v", t))
	}
}
func (r *rest) FieldQuery(name string, fq FQ) {
	r.registerType(fq.Type)
	r.registerQuery(name, fq)
}
func (r *rest) SelectorQuery(name string, sq SQ) {
	r.registerType(sq.BodyType)
	r.registerType(sq.ResultType)
	r.registerQuery(name, sq)
}
func (r *rest) CustomQuery(name string, cq CQ) {
	r.registerType(cq.BodyType)
	r.registerType(cq.ResultType)
	r.registerQuery(name, cq)
}
func (r *rest) Index(typ interface{}, index Index) {
	r.registerType(typ)
	c := r.s.DB(r.db).C(r.typeName(typ))
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

func (r *rest) queryRes(query interface{}, uri *URI, ctx *Context) (res Resource, err error) {
	panic("Not Implement")
}
func (r *rest) R(uri *URI, ctx *Context) (res Resource, err error) {
	name := uri.Path[0]
	if typ, ok := r.types[name]; ok {
		return r.typeRes(typ, uri, ctx)
	}
	if qry, ok := r.queries[name]; ok {
		if isSysQueryName(name) && !ctx.Sys {
			return nil, &RESTError{Code: Forbidden, Msg: uri.String()}
		}
		return r.queryRes(qry, uri, ctx)
	}
	return nil, &RESTError{Code: NotFound, Msg: uri.String()}
}
