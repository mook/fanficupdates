package util

import (
	"bytes"
	"encoding/base64"
	"io"
	"math/rand"
	"time"
)

type RandomStringReader struct {
	reader  io.Reader
	encoder io.WriteCloser
	buffer  bytes.Buffer
}

func NewRandomStringReader() *RandomStringReader {
	result := &RandomStringReader{
		reader: rand.New(rand.NewSource(time.Now().Unix())),
	}
	result.encoder = base64.NewEncoder(base64.StdEncoding, &result.buffer)
	return result
}

func (r *RandomStringReader) Read(p []byte) (int, error) {
	required := len(p) - r.buffer.Len()
	if required > 0 {
		// Determine number of raw bytes to read, rounding up to 4
		count := int64((required + 3) / 4 * 3)
		io.CopyN(r.encoder, r.reader, count)
	}
	return r.buffer.Read(p)
}

func (r *RandomStringReader) ReadString(length int) (string, error) {
	buf := make([]byte, length)
	_, err := r.Read(buf)
	if err != nil {
		return "", err
	}
	return string(buf), nil
}

func (r *RandomStringReader) MustReadString(length int) string {
	result, err := r.ReadString(length)
	if err != nil {
		panic(err)
	}
	return result
}

func (r *RandomStringReader) MustReadStringSlice(size, length int) []string {
	count := rand.Intn(size)
	var result []string
	for i := 0; i < count; i++ {
		result = append(result, r.MustReadString(length))
	}
	return result
}

func RandomTime() time.Time {
	loc, err := time.LoadLocation("")
	if err != nil {
		panic(err)
	}
	return time.Now().In(loc).Add(time.Duration(int64(rand.Uint64()))).Round(time.Second)
}
