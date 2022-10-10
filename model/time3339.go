package model

import (
	"strings"
	"time"
)

type Time3339 struct {
	time.Time
}

func (t Time3339) String() string {
	return t.Format(time.RFC3339)
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
