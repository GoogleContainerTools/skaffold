package pkging

import (
	"encoding/json"
	"time"
)

const timeFmt = time.RFC3339Nano

type ModTime time.Time

func (m ModTime) MarshalJSON() ([]byte, error) {
	t := time.Time(m)
	return json.Marshal(t.Format(timeFmt))
}

func (m *ModTime) UnmarshalJSON(b []byte) error {
	t := time.Time{}
	if err := json.Unmarshal(b, &t); err != nil {
		return err
	}
	(*m) = ModTime(t)
	return nil
}
