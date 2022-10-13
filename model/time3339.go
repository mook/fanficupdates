package model

import (
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
	result, err := time.Parse(time.RFC3339, input)
	if err != nil {
		return err
	}
	t.Time = result
	return nil
}
