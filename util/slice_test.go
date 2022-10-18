package util_test

import (
	"fmt"
	"testing"

	"github.com/mook/fanficupdates/util"
	"github.com/stretchr/testify/assert"
)

func TestMap(t *testing.T) {
	t.Run("identity", func(t *testing.T) {
		input := []int{1, 2, 3, 4, 5}
		output := util.Map(input, func(i int) int { return i })
		assert.Equal(t, input, output)
	})
	t.Run("transform", func(t *testing.T) {
		input := []int{1, 2, 3, 4, 5}
		expected := []int{2, 3, 4, 5, 6}
		output := util.Map(input, func(i int) int { return i + 1 })
		assert.Equal(t, expected, output)
		assert.Equal(t, []int{1, 2, 3, 4, 5}, input, "modified the input")
	})
	t.Run("type change", func(t *testing.T) {
		input := []int{1, 2, 3, 4, 5}
		expected := []string{"1", "2", "3", "4", "5"}
		output := util.Map(input, func(i int) string { return fmt.Sprintf("%d", i) })
		assert.Equal(t, expected, output)
	})
}

func TestAny(t *testing.T) {
	t.Run("hit", func(t *testing.T) {
		input := []int{1, 2, 3, 4, 5}
		result := util.Any(input, func(i int) bool { return i == 4 })
		assert.True(t, result)
	})
	t.Run("miss", func(t *testing.T) {
		input := []int{1, 2, 3, 4, 5}
		result := util.Any(input, func(i int) bool { return i == 7 })
		assert.False(t, result)
	})
}

func TestFind(t *testing.T) {
	t.Run("hit", func(t *testing.T) {
		input := []int{1, 2, 3, 4, 5}
		result := util.Find(input, func(i int) bool { return i == 3 })
		if assert.Equal(t, &input[2], result) {
			assert.Equal(t, *result, 3)
		}
	})
	t.Run("miss", func(t *testing.T) {
		input := []int{1, 2, 3, 4, 5}
		result := util.Find(input, func(i int) bool { return i == 7 })
		assert.Nil(t, result)
	})
	t.Run("first", func(t *testing.T) {
		input := []int{1, 2, 3, 4, 5, 1, 2, 3, 4, 5}
		result := util.Find(input, func(i int) bool { return i == 3 })
		if assert.Equal(t, &input[2], result) {
			assert.Equal(t, *result, 3)
		}
	})
}

func TestFilter(t *testing.T) {
	input := []int{1, 2, 3, 4, 5}
	result := util.Filter(input, func(i int) bool { return i%2 == 0 })
	assert.Equal(t, []int{2, 4}, result)
	assert.Equal(t, []int{1, 2, 3, 4, 5}, input, "modified the input")
}
