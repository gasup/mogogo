package net

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mogogo"
	"net/http"
	"reflect"
	"runtime/debug"
	"strconv"
	"strings"
	"time"
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
	PrefetchConfig mogogo.M
	s              mogogo.Session
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
			return nil, &mogogo.Error{Code: mogogo.BadRequest, Msg: "parse json error", Err: err}
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
func (h *HTTPHandler) requestForPrefetch(urlStr string, ctx *mogogo.Context, cfg mogogo.M) (ret map[string]interface{}) {
	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		panic(&mogogo.Error{Code: mogogo.InternalServerError, Err: err})
	}
	status, r := h.request(req, ctx, cfg, false)
	m, ok := r.(map[string]interface{})
	if !ok {
		panic(&mogogo.Error{
			Code: mogogo.InternalServerError,
			Msg:  fmt.Sprintf("prefetch only support json, but %T", r),
		})
	}
	if status >= 500 {
		panic(&mogogo.Error{
			Code: mogogo.InternalServerError,
			Msg:  fmt.Sprintf("%v", m["statusMsg"]),
		})
	}
	ret = m
	return
}
func (h *HTTPHandler) prefetchField(req *http.Request, ctx *mogogo.Context, val interface{}, cfg mogogo.M) (ret interface{}) {
	switch t := val.(type) {
	case map[string]interface{}:
		if href, ok := t["href"]; ok {
			m := h.requestForPrefetch(href.(string), ctx, cfg)
			m["href"] = href
			ret = m
		} else {
			ret = val
		}
	default:
		ret = val
	}
	return
}
func (h *HTTPHandler) prefetch(req *http.Request, ctx *mogogo.Context, m map[string]interface{}, cfg mogogo.M) {
	if cfg == nil {
		return
	}

	for f, v := range cfg {
		if f[0] == '$' {
			continue
		}
		fv, ok := m[f]
		if !ok {
			continue
		}
		var fieldcfg mogogo.M = nil
		if v != nil {
			fieldcfg, ok = v.(mogogo.M)
			if !ok {
				panic(&mogogo.Error{
					Code: mogogo.InternalServerError,
					Msg:  fmt.Sprintf("'%s' want type mogogo.M", f),
				})
			}
			hidden := getBool(fieldcfg, "$hidden")
			if hidden {
				delete(m, f)
			} else {
				m[f] = h.prefetchField(req, ctx, fv, fieldcfg)
			}
		} else {
			m[f] = h.prefetchField(req, ctx, fv, nil)
		}
	}
}
func getBool(m mogogo.M, key string) (ret bool) {
	if m == nil {
		return false
	}
	val, ok := m[key]
	if ok {
		switch t := val.(type) {
		case bool:
			ret = t
		case int:
			switch t {
			case 0:
				ret = false
			case 1:
				ret = true
			default:
				panic(&mogogo.Error{
					Code: mogogo.InternalServerError,
					Msg:  fmt.Sprintf("'%s' want type bool, got %d", key, t),
				})
			}
		default:
			panic(&mogogo.Error{
				Code: mogogo.InternalServerError,
				Msg:  fmt.Sprintf("'%s' want type bool, got %v", key, t),
			})
		}
	} else {
		ret = false
	}
	return
}
func (h *HTTPHandler) responseToMap(req *http.Request, ctx *mogogo.Context, rm mogogo.ResourceMeta, r interface{}, cfg mogogo.M, start bool) map[string]interface{} {
	ret := rm.ResponseToMap(r, req.URL)
	var norels bool
	if start {
		norels, _ = strconv.ParseBool(req.URL.Query().Get("norels"))
	} else {
		if cfg == nil {
			norels = true
		} else if _, ok := cfg["$norels"]; ok {
			norels = getBool(cfg, "$norels")
		} else {
			norels = true
		}
	}
	if !norels {
		base, ok := getBase(r)
		if ok {
			for n, rid := range base.AllRels() {
				ret[strings.ToLower(n)] = map[string]interface{}{"href": rid.URLWithBase(req.URL).String()}
			}
		}
	}
	h.prefetch(req, ctx, ret, cfg)
	return ret
}
func (h *HTTPHandler) responseIter(req *http.Request, ctx *mogogo.Context, iter mogogo.Iter, rm mogogo.ResourceMeta, cfg mogogo.M, start bool) (status int, resp interface{}) {
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
			i := h.responseToMap(req, ctx, rm, v, cfg, start)
			items = append(items, i)
		}
		m["slice"] = items
		if len(items) == 0 {
			status = 404
		}
	}
	m["statusCode"] = status
	return
}
func (h *HTTPHandler) responseBody(req *http.Request, ctx *mogogo.Context, r interface{}, res mogogo.Resource, cfg mogogo.M, start bool) (status int, resp interface{}) {
	resMeta := res.(mogogo.ResourceMeta)
	switch t := r.(type) {
	case mogogo.Iter:
		status, resp = h.responseIter(req, ctx, t, resMeta, cfg, start)
	default:
		if r == nil {
			status = 200
			resp = map[string]interface{}{"statusCode": status}
		} else {
			m := h.responseToMap(req, ctx, resMeta, r, cfg, start)
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
func (h *HTTPHandler) paramsFromConfig(resId *mogogo.ResId, cfg mogogo.M) {
	if cfg == nil {
		return
	}
	if n, ok := cfg["$n"]; ok {
		resId.Params["n"] = fmt.Sprintf("%v", n)
	} else if all, ok := cfg["$all"]; ok {
		resId.Params["all"] = fmt.Sprintf("%v", all)
	} else if noitems, ok := cfg["$noitems"]; ok {
		resId.Params["noitems"] = fmt.Sprintf("%v", noitems)
	}
}
func (h *HTTPHandler) request(req *http.Request, ctx *mogogo.Context, cfg mogogo.M, start bool) (status int, resp interface{}) {
	resId, err := mogogo.ResIdFromURL(req.URL)
	if err != nil {
		return h.errToMap(err)
	}
	res, err := h.s.R(resId, ctx)
	if err != nil {
		return h.errToMap(err)
	}
	if start {
		var ok bool
		cfg, ok = h.PrefetchConfig[resId.Name()].(mogogo.M)
		if !ok {
			rm := res.(mogogo.ResourceMeta)
			cfg, ok = h.PrefetchConfig[rm.ResponseType().Name()].(mogogo.M)
		}
	}
	h.paramsFromConfig(res.Id(), cfg)
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
	status, resp = h.responseBody(req, ctx, r, res, cfg, start)
	return
}
func (h *HTTPHandler) compress(rw http.ResponseWriter, req *http.Request, m map[string]interface{}) (*bytes.Buffer, error) {
	buf, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	ret := bytes.NewBuffer(make([]byte, 0, 512))
	var w io.Writer
	ae := req.Header.Get("Accept-Encoding")
	if strings.Contains(ae, "gzip") {
		rw.Header().Set("Content-Encoding", "gzip")
		gw := gzip.NewWriter(ret)
		defer gw.Close()
		w = gw
	} else if strings.Contains(ae, "deflate") {
		rw.Header().Set("Content-Encoding", "deflate")
		fw, _ := flate.NewWriter(ret, flate.DefaultCompression)
		defer fw.Close()
		w = fw
	} else {
		w = ret
	}
	_, err = w.Write(buf)
	if err != nil {
		return nil, err
	}
	return ret, nil
}
func (h *HTTPHandler) log(w http.ResponseWriter, req *http.Request, status int, msg string, startTime time.Time) {
	ip := req.Header.Get("X-Forwarded-For")
	if ip == "" {
		ip = req.RemoteAddr
	}
	ctxId := ""
	if c, err := req.Cookie(cookieKey); err == nil {
		ctxId = c.Value[0:12]
	}
	s := ""
	if msg != "" {
		s = " - "
	}
	elapsed := time.Now().Sub(startTime)
	log.Printf("%s \"%s\" %d \"%s\" \"%s\" %v%s%s\n", req.Method, req.URL.RequestURI(), status, ctxId, ip, elapsed, s, msg)
}
func (h *HTTPHandler) logMap(w http.ResponseWriter, req *http.Request, status int, m map[string]interface{}, startTime time.Time) {
	msg := ""
	if status >= 400 {
		if sm, ok := m["statusMsg"]; ok {
			msg = sm.(string)
		}
		if stack, ok := m["stackTrace"]; ok {
			msg = "! " + msg
			msg = fmt.Sprintf("%s\n ! %s", msg, strings.Join(stack.([]string), "\n ! "))
		}
	}
	h.log(w, req, status, msg, startTime)
}
func (h *HTTPHandler) responseJSON(w http.ResponseWriter, req *http.Request, status int, m map[string]interface{}, startTime time.Time) {
	if m == nil {
		w.WriteHeader(status)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "private, max-age=0")
	buf, err := h.compress(w, req, m)
	if err != nil {
		h.responseError(w, req, err, "", startTime)
		return
	}
	me := req.Header.Get("If-None-Match")
	et := etag(buf.Bytes())
	w.Header().Set("Etag", et)
	if me == et {
		w.WriteHeader(304)
	} else {
		w.WriteHeader(status)
		_, err = buf.WriteTo(w)
		if err != nil {
			log.Printf("JSON ENCODE ERROR: %v\n", err)
		}
	}
	h.logMap(w, req, status, m, startTime)
}
func (h *HTTPHandler) responseError(w http.ResponseWriter, req *http.Request, err interface{}, stack string, startTime time.Time) {
	s, m := h.errToMap(err)
	if stack != "" {
		m["stackTrace"] = strings.Split(stack, "\n")
	}
	h.responseJSON(w, req, s, m, startTime)
}

const (
	cookieKey     = "MOGOGO_ID"
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
		if time.Since(ts) > 24*time.Hour {
			expires := time.Now().Add(365 * 24 * time.Hour)
			c.Path = "/"
			c.Expires = expires
			http.SetCookie(w, c)
			http.SetCookie(w, &http.Cookie{
				Name:    cookieTimeKey,
				Value:   strconv.FormatInt(time.Now().Unix(), 36),
				Path:    "/",
				Expires: expires,
			})
		}
	}
}
func (h *HTTPHandler) storeContext(ctxId string, w http.ResponseWriter, req *http.Request, ctx *mogogo.Context) {
	if h.ContextHandler == nil {
		return
	}
	if ctxId == "" {
		ctxId = randId()
		http.SetCookie(w, &http.Cookie{
			Name:    cookieKey,
			Value:   ctxId,
			Path:    "/",
			Expires: time.Now().Add(365 * 24 * time.Hour),
		})
		http.SetCookie(w, &http.Cookie{
			Name:    cookieTimeKey,
			Value:   strconv.FormatInt(time.Now().Unix(), 36),
			Path:    "/",
			Expires: time.Now().Add(365 * 24 * time.Hour),
		})
	}
	if ctx.IsUpdated() {
		h.ContextHandler.Store(ctxId, ctx, req)
	}
	h.updateCookieExpires(w, req)
}
func (h *HTTPHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	startTime := time.Now()
	req.URL.Host = req.Host
	if req.TLS == nil {
		req.URL.Scheme = "http"
	} else {
		req.URL.Scheme = "https"
	}
	defer func() {
		err := recover()
		if err != nil {
			h.responseError(w, req, err, string(debug.Stack()), startTime)
		}
	}()
	ctx := h.s.NewContext()
	defer ctx.Close()
	ctxId := h.loadContext(req, ctx)
	status, resp := h.request(req, ctx, nil, true)
	h.storeContext(ctxId, w, req, ctx)
	if h.ContextHandler != nil {
	}
	switch t := resp.(type) {
	case map[string]interface{}:
		h.responseJSON(w, req, status, t, startTime)
	default:
		if t != nil {
			h.responseError(w, req, fmt.Sprintf("unexpected response type '%T'", t), "", startTime)
		} else {
			h.responseJSON(w, req, status, nil, startTime)
		}
	}

}
func NewHTTPHandler(s mogogo.Session) *HTTPHandler {
	if s == nil {
		panic("param 's' is null")
	}
	return &HTTPHandler{s: s}
}
