package net

import (
	"crypto/rand"
	"encoding/base32"
	"hash/crc64"
	"io"
	"strconv"
)

var crc64Table = crc64.MakeTable(crc64.ISO)

func etag(b []byte) string {
	sum := crc64.Checksum(b, crc64Table)
	return strconv.FormatUint(sum, 36)
}
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
