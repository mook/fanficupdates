package util_test

import (
	"fmt"
	"testing"

	"github.com/mook/fanficupdates/util"
)

func TestRandomStringReader(t *testing.T) {
	r := util.NewRandomStringReader()
	sizes := []int{
		7, 1, 5, 0, 4, 9,
	}
	for _, size := range sizes {
		t.Run(fmt.Sprintf("%d", size), func(t *testing.T) {
			buf := make([]byte, size)
			n, err := r.Read(buf)
			if err != nil {
				t.Errorf("failed to read %d bytes: %v", size, err)
			}
			if n != size {
				t.Errorf("expected to read %d bytes, got %d", size, n)
			}
			result := string(buf)
			if len(result) != size {
				t.Errorf("expected to read %d bytes, got %d (%s)", size, len(result), result)
			}
		})
	}
}

func TestReadString(t *testing.T) {
	r := util.NewRandomStringReader()
	sizes := []int{
		7, 1, 5, 0, 4, 9,
	}
	for _, size := range sizes {
		str, err := r.ReadString(size)
		if err != nil {
			t.Errorf("failed to read %d bytes: %v", size, err)
		}
		if len(str) != size {
			t.Errorf("wanted to read %d bytes, got %d: %s", size, len(str), str)
		}
	}
}
