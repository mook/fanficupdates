package assertx

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/mook/fanficupdates/util"
	"github.com/stretchr/testify/assert"
)

type testingT interface {
	Errorf(format string, args ...interface{})
	Helper()
}

func formatSlice[T any](slice []T) string {
	if len(slice) < 1 {
		return fmt.Sprintf("%#v", slice)
	}
	v := reflect.ValueOf(slice[0])
	kind := v.Kind()
	if kind != reflect.Interface && kind != reflect.Pointer {
		return fmt.Sprintf("%#v", slice)
	}
	var results []string
	for _, item := range slice {
		v = reflect.ValueOf(item).Elem()
		results = append(results, fmt.Sprintf("%#v", v))
	}
	return fmt.Sprintf("[%s]", strings.Join(results, ","))
}

func Any[T any](t testingT, slice []T, predicate func(input T) bool, msgAndArgs ...any) bool {
	t.Helper()
	if util.Any(slice, predicate) {
		return true
	}
	return assert.Fail(t, fmt.Sprintf("%s does not contain acceptable elements", formatSlice(slice)), msgAndArgs...)
}
