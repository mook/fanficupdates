package model

import (
	"errors"
	"strings"
	"time"
)

type Time3339 struct {
	time.Time
}

// NewTime1339 returns a new Time3339.  If the given time is zero time, use the
// current time instead.
func NewTime3339(t time.Time) *Time3339 {
	if t.IsZero() {
		t = time.Now()
	}
	return &Time3339{Time: t.Round(time.Second)}
}

func (t *Time3339) String() string {
	return t.Round(time.Second).Format(time.RFC3339)
}

func (t *Time3339) UnmarshalJSON(bytes []byte) error {
	input := strings.Trim(string(bytes), `"`)
	if input == "null" {
		return nil
	}
	formats := []string{
		time.RFC3339,
		time.RFC1123,
		time.RubyDate,
		time.UnixDate,
		time.ANSIC,
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02",
		"2006/01/02",
		"Mon Jan 2 2006",
		"Jan 2 2006",
		"1/2/2006",
	}
	var parseError *time.ParseError
	for _, format := range formats {
		result, err := time.Parse(format, input)
		if err == nil {
			t.Time = result
			return nil
		}
		if !errors.As(err, &parseError) {
			return err
		}
	}
	return &time.ParseError{
		Value:   input,
		Message: ": failed to parse as any of the known formats",
	}
}
