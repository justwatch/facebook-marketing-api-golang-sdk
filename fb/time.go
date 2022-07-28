package fb

import (
	"encoding/json"
	"time"
)

// Time is used since the timestamp format used by the Graph API is not 100%
// the one used for unmarshaling time fields by the encoding/json Go package.
type Time time.Time

const tsFormat = "2006-01-02T15:04:05-0700"

// UnmarshalJSON implements json.Unmarshaler.
func (t *Time) UnmarshalJSON(data []byte) error {
	var s string
	err := json.Unmarshal(data, &s)
	if err != nil {
		return err
	} else if s == "" {
		return nil
	}

	ts, err := time.Parse(tsFormat, s)
	if err != nil {
		return err
	}
	*t = Time(ts)

	return nil
}

// MarshalJSON implements json.Marshaler.
func (t Time) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Time(t).Format(tsFormat))
}
