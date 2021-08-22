package main

import (
	"time"
)

// PTime is a simple time.Time wrapped for a more human readable json representation of "2006-01-02 15:04:05"
type PTime struct {
	time.Time
}

// MarshalJSON as the name implies
func (pt *PTime) MarshalJSON() ([]byte, error) {
	b := []byte("\"" + pt.Time.Format("2006-01-02 15:04:05") + "\"")
	return b, nil
}

// UnmarshalJSON as the name implies
func (pt *PTime) UnmarshalJSON(b []byte) (err error) {
	s := string(b)

	t, err := time.Parse("\"2006-01-02 15:04:05\"", s)
	if err != nil {
		return err
	}
	pt.Time = t

	return nil
}
