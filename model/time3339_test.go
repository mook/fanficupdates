package model_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/mook/fanficupdates/model"
	"github.com/stretchr/testify/assert"
)

func TestNewTime3339(t *testing.T) {
	t.Run("fixed", func(t *testing.T) {
		input := time.Date(1234, 5, 6, 7, 8, 9, 10, time.UTC)
		// Expect to truncate to seconds
		expected := time.Date(1234, 5, 6, 7, 8, 9, 0, time.UTC)
		result := model.NewTime3339(input)
		assert.Equal(t, expected, result.Time)
	})
	t.Run("zero", func(t *testing.T) {
		result := model.NewTime3339(time.Time{})
		assert.NotZero(t, result.Time)
		assert.Zero(t, result.Time.Nanosecond())
	})
}

func TestString(t *testing.T) {
	input := time.Date(1234, 5, 6, 7, 8, 9, 10, time.UTC)
	result := model.NewTime3339(input).String()
	assert.Equal(t, "1234-05-06T07:08:09Z", result)
}

func TestUnmarshalJSON(t *testing.T) {
	t.Run("null", func(t *testing.T) {
		subject := model.Time3339{
			Time: time.Date(1234, 5, 6, 7, 8, 9, 10, time.UTC),
		}
		if assert.NoError(t, subject.UnmarshalJSON([]byte("null"))) {
			expected := time.Date(1234, 5, 6, 7, 8, 9, 10, time.UTC)
			assert.Equal(t, expected, subject.Time)
		}
	})
	t.Run("success", func(t *testing.T) {
		subject := model.Time3339{}
		input := []byte(`"1234-05-06T07:08:09Z"`)
		expected := time.Date(1234, 5, 6, 7, 8, 9, 0, time.UTC)
		if assert.NoError(t, json.Unmarshal(input, &subject)) {
			assert.Equal(t, expected, subject.Time)
		}
	})
	t.Run("error", func(t *testing.T) {
		subject := model.Time3339{}
		input := []byte(`"not a valid time"`)
		assert.Error(t, json.Unmarshal(input, &subject))
	})
}
