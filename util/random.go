package util

import (
	"bytes"
	"encoding/base64"
	"io"
	"math/rand"
	"time"
)

var randReader io.Reader
var randEncoder io.Writer
var randBuffer bytes.Buffer

func init() {
	randReader = rand.New(rand.NewSource(time.Now().Unix()))
	randEncoder = base64.NewEncoder(base64.URLEncoding, &randBuffer)
}

// RandomStringWithLength returns a random string with a given length
func RandomStringWithLength(length int) string {
	required := length - randBuffer.Len()
	if required > 0 {
		// Determine number of raw bytes to read, rounding up to 4
		count := int64((required + 3) / 4 * 3)
		_, err := io.CopyN(randEncoder, randReader, count)
		if err != nil {
			panic(err)
		}
	}
	return string(randBuffer.Next(length))
}

// RandomString returns a random non-empty string of an unspecified length
func RandomString() string {
	return RandomStringWithLength(rand.Intn(31) + 1)
}

// RandomList returns a random list with a length no more than specified, where
// each element is generated with the given function.
func RandomList[T any](maxLength int, gen func() T) []T {
	count := rand.Intn(maxLength)
	result := make([]T, 0, count)
	for i := 0; i < count; i++ {
		result = append(result, gen())
	}
	return result
}

// RandomTime returns a random time
func RandomTime() time.Time {
	loc, err := time.LoadLocation("")
	if err != nil {
		panic(err)
	}
	return time.Now().In(loc).Add(time.Duration(int64(rand.Uint64()))).Round(time.Second)
}
