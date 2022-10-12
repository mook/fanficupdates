package util_test

import (
	"testing"

	"github.com/mook/fanficupdates/util"
	"github.com/stretchr/testify/assert"
)

func FuzzRandomStringWithLength(f *testing.F) {
	for _, length := range []int{7, 1, 5, 0, 4, 9} {
		f.Add(length)
	}
	f.Fuzz(func(t *testing.T, size int) {
		result := util.RandomStringWithLength(size)
		assert.Len(t, result, size)
	})
}

func TestRandomString(t *testing.T) {
	result := util.RandomString()
	assert.Greater(t, len(result), 0)
	assert.LessOrEqual(t, len(result), 32)
}

func FuzzRandomList(f *testing.F) {
	for _, maxLength := range []int{0, 1, 32} {
		f.Add(maxLength)
	}
	f.Fuzz(func(t *testing.T, maxLength int) {
		count := 0
		result := util.RandomList(maxLength+1, func() string {
			count++
			return ""
		})
		assert.Len(t, result, count)
		assert.LessOrEqual(t, count, maxLength+1)
	})
}

func TestRandomTime(t *testing.T) {
	result := util.RandomTime()
	assert.NotZero(t, result)
	assert.Zero(t, result.Nanosecond())
}
