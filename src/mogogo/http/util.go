package http

import (
	"crypto/rand"
	"io"
	"encoding/base32"
)

func randId() string {
	n := 20
	b := make([]byte, n)
	n, err := io.ReadFull(rand.Reader, b)
	if n != len(b) {
		panic("random bytes not enough")
	}
	if err != nil {
		panic(err)
	}
	return base32.HexEncoding.EncodeToString(b)
}

