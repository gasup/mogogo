package mogogo

import (
	"io"
	"net/http"
)

type httpHandler struct {
	s Session
}

func (h *httpHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	io.WriteString(w, "hello, world!\n")
}

func NewHTTPHandler(s Session) http.Handler {
	return &httpHandler{s: s}
}
