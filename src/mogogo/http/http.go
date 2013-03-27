package http

import (
	"encoding/json"
	"fmt"
	"log"
	"mogogo"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"
	"compress/gzip"
	"compress/flate"
	"io"
)

func getBase(s interface{}) (base *mogogo.Base, ok bool) {
	fv := reflect.ValueOf(s).Elem().FieldByName("Base")
	if fv.IsValid() {
		base, ok = fv.Addr().Interface().(*mogogo.Base), true
	} else {
		base, ok = nil, false
	}
	return
}
type ContextHandler interface {
	Load(ctxId string, ctx *mogogo.Context, req *http.Request)
	Store(ctxId string, ctx *mogogo.Context, req *http.Request)
}
type HTTPHandler struct {
	ContextHandler ContextHandler
	s mogogo.Session
}

func (h *HTTPHandler) mggErrToMap(err *mogogo.Error) (status int, m map[string]interface{}) {
	m = make(map[string]interface{})
	status = int(err.Code)
	m["statusCode"] = status
	m["statusMsg"] = err.Error()
	if err.Fields != nil {
		m["fields"] = err.Fields
	}
	return
}
func (h *HTTPHandler) errToMap(err interface{}) (status int, m map[string]interface{}) {
	switch t := err.(type) {
	case *mogogo.Error:
		status, m = h.mggErrToMap(t)
	case error:
		status, m = h.mggErrToMap(&mogogo.Error{Code: mogogo.InternalServerError, Err: t})
	default:
		msg := fmt.Sprintf("%v", t)
		status, m = h.mggErrToMap(&mogogo.Error{Code: mogogo.InternalServerError, Msg: msg})
	}
	return
}
func (h *HTTPHandler) requestBody(req *http.Request, res mogogo.Resource) (body interface{}, err error) {
	resMeta := res.(mogogo.ResourceMeta)
	ct := req.Header.Get("Content-Type")
	if ct == "application/json" {
		var m map[string]interface{}
		dec := json.NewDecoder(req.Body)
		err = dec.Decode(&m)
		if err != nil {
			return nil, err
		}
		if req.Method == "PATCH" {
			body, err = resMeta.MapToUpdater(m, req.URL)
		} else {
			body, err = resMeta.MapToRequest(m, req.URL)
		}
	} else {
		body, err = nil, &mogogo.Error{Code: mogogo.UnsupportedMediaType}
	}
	return
}
func (h *HTTPHandler) responseToMap(req *http.Request, rm mogogo.ResourceMeta, r interface{}) map[string]interface{} {
	ret := rm.ResponseToMap(r, req.URL)
	norels, _ := strconv.ParseBool(req.URL.Query().Get("norels"))
	if norels {
		return ret
	}
	base, ok := getBase(r)
	if !ok {
		return ret
	}
	for n, rid := range base.AllRels() {
		ret[strings.ToLower(n)] = map[string]interface{} {"self":rid.URLWithBase(req.URL).String()}
	}
	return ret
}
func (h *HTTPHandler) responseIter(req *http.Request, iter mogogo.Iter, rm mogogo.ResourceMeta) (status int, resp interface{}) {
	s, err := iter.Slice()
	if err != nil {
		return h.errToMap(err)
	}
	m := make(map[string]interface{})
	resp = m
	status = 200
	m["self"] = s.Self().URLWithBase(req.URL).String()
	if s.HasPrev() {
		m["prev"] = s.Prev().URLWithBase(req.URL).String()
	}
	if s.HasNext() {
		m["next"] = s.Next().URLWithBase(req.URL).String()
	}
	if s.HasCount() {
		m["count"] = s.Count()
		m["more"] = s.More()
	}
	if s.HasItems() {
		items := make([]interface{}, 0, len(s.Items()))
		for _, v := range s.Items() {
			i := h.responseToMap(req, rm, v)
			items = append(items, i)
		}
		m["items"] = items
		if len(items) == 0 {
			status = 404
		}
	}
	m["statusCode"] = status
	return
}
func (h *HTTPHandler) responseBody(req *http.Request, r interface{}, res mogogo.Resource) (status int, resp interface{}) {
	resMeta := res.(mogogo.ResourceMeta)
	switch t := r.(type) {
	case mogogo.Iter:
		status, resp = h.responseIter(req, t, resMeta)
	default:
		if r == nil {
			status = 200
			resp = map[string]interface{}{"statusCode": status}
		} else {
			m := h.responseToMap(req, resMeta, r)
			if base, ok := getBase(r); ok && base.IsNew() {
				status = 201
			} else {
				status = 200
			}
			m["statusCode"] = status
			resp = m
		}
	}
	return
}
func (h *HTTPHandler) request(req *http.Request, ctx *mogogo.Context) (status int, resp interface{}) {
	resId, err := mogogo.ResIdFromURL(req.URL)
	if err != nil {
		return h.errToMap(err)
	}
	res, err := h.s.R(resId, ctx)
	if err != nil {
		return h.errToMap(err)
	}
	var r interface{}
	var body interface{}
	switch req.Method {
	case "GET":
		r, err = res.Get()
	case "PUT":
		body, err = h.requestBody(req, res)
		if err != nil {
			return h.errToMap(err)
		}
		r, err = res.Put(body)
	case "DELETE":
		r, err = res.Delete()
	case "POST":
		body, err = h.requestBody(req, res)
		if err != nil {
			return h.errToMap(err)
		}
		r, err = res.Post(body)
	case "PATCH":
		body, err = h.requestBody(req, res)
		if err != nil {
			return h.errToMap(err)
		}
		r, err = res.Patch(body)
	default:
		return h.errToMap(&mogogo.Error{Code: mogogo.MethodNotAllowed})
	}
	if err != nil {
		return h.errToMap(err)
	}
	return h.responseBody(req, r, res)
}
func (h *HTTPHandler) compress(w http.ResponseWriter, req *http.Request) (cw io.WriteCloser, compressed bool) {
	ae := req.Header.Get("Accept-Encoding")
	if strings.Contains(ae, "gzip") {
		w.Header().Set("Content-Encoding", "gzip")
		cw, compressed = gzip.NewWriter(w), true
	} else if strings.Contains(ae, "deflate") {
		w.Header().Set("Content-Encoding", "deflate")
		cw, _ = flate.NewWriter(w, flate.DefaultCompression)
		compressed = true
	} else {
		cw, compressed = nil, false
	}
	return
}
func (h *HTTPHandler) responseJSON(w http.ResponseWriter, req *http.Request, status int, m map[string]interface{}) {
	if m == nil {
		w.WriteHeader(status)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	cw, c := h.compress(w, req)
	if c {
		defer cw.Close()
	}
	w.WriteHeader(status)
	var enc *json.Encoder
	if c {
		enc = json.NewEncoder(cw)
	} else {
		enc = json.NewEncoder(w)
	}
	err := enc.Encode(m)
	if err != nil {
		log.Println(err)
	}
}
func (h *HTTPHandler) responseError(w http.ResponseWriter, req *http.Request, err interface{}) {
	s, m := h.errToMap(err)
	h.responseJSON(w, req, s, m)
}
const (
	cookieKey = "MOGOGO_ID"
	cookieTimeKey = "MOGOGO_TS"
)
func (h *HTTPHandler) loadContext(req *http.Request, ctx *mogogo.Context) (ctxId string) {
	if h.ContextHandler == nil {
		return
	}
	if c, err := req.Cookie(cookieKey); err == nil {
		ctxId = c.Value
		h.ContextHandler.Load(ctxId, ctx, req)
	}
	ctx.SetUpdated(false)
	return
}
func (h *HTTPHandler) updateCookieExpires(w http.ResponseWriter, req *http.Request) {
	if c, err := req.Cookie(cookieKey); err == nil {
		var ts time.Time
		cts, err := req.Cookie(cookieTimeKey)
		if err == nil {
			unix, err := strconv.ParseInt(cts.Value, 36, 64)
			if err == nil {
				ts = time.Unix(unix, 0)
			} else {
				ts = time.Unix(0, 0)
			}
		} else {
			ts = time.Unix(0, 0)
		}
		if time.Since(ts) > 24 * time.Hour {
			expires := time.Now().Add(365 * 24 * time.Hour)
			c.Expires = expires
			http.SetCookie(w, c)
			http.SetCookie(w, &http.Cookie{
				Name: cookieTimeKey,
				Value: strconv.FormatInt(time.Now().Unix(), 36),
				Expires: expires,
			})
		}
	}
}
func (h *HTTPHandler) storeContext(ctxId string, w http.ResponseWriter, req *http.Request, ctx *mogogo.Context) {
	if h.ContextHandler == nil {
		return
	}
	if ctx.IsUpdated() {
		if ctxId == "" {
			ctxId = randId()
			http.SetCookie(w, &http.Cookie{
				Name: cookieKey,
				Value: ctxId,
				Expires: time.Now().Add(365 * 24 * time.Hour),
			})
			http.SetCookie(w, &http.Cookie{
				Name: cookieTimeKey,
				Value: strconv.FormatInt(time.Now().Unix(), 36),
				Expires: time.Now().Add(365 * 24 * time.Hour),
			})
		}
		h.ContextHandler.Store(ctxId, ctx, req)
	}
	h.updateCookieExpires(w, req)
}
func (h *HTTPHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	req.URL.Host = req.Host
	if req.TLS == nil {
		req.URL.Scheme = "http"
	} else {
		req.URL.Scheme = "https"
	}
	defer func() {
		err := recover()
		if err != nil {
			h.responseError(w, req, err)
		}
	}()
	ctx := h.s.NewContext()
	defer ctx.Close()
	ctxId := h.loadContext(req, ctx)
	status, resp := h.request(req, ctx)
	h.storeContext(ctxId, w, req, ctx)
	if h.ContextHandler != nil {
	}
	switch t := resp.(type) {
	case map[string]interface{}:
		h.responseJSON(w, req, status, t)
	default:
		if t != nil {
			h.responseError(w, req, fmt.Sprintf("unexpected response type '%T'", t))
		} else {
			h.responseJSON(w, req, status, nil)
		}
	}

}
func NewHandler(s mogogo.Session) *HTTPHandler {
	if s == nil {
		panic("'s' is null")
	}
	return &HTTPHandler{s: s}
}
