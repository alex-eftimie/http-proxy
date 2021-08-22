package main

import "time"

// Duration represents a duration
type Duration struct {
	time.Duration
}

// MarshalJSON .
func (d Duration) MarshalJSON() ([]byte, error) {
	return []byte("\"" + d.String() + "\""), nil
}

// UnmarshalJSON .
func (d *Duration) UnmarshalJSON(b []byte) error {
	s := string(b)
	s = s[1 : len(s)-1]
	t, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	d.Duration = t
	return nil
}

// Adds parses d2 and adds the duration to d
func (d *Duration) Adds(d2 string) {
	d2d, _ := ParseDuration(d2)
	d.Add(d2d)
}

// Add returns d2 added to d
func (d *Duration) Add(d2 Duration) {
	d.Duration = time.Duration(d.Duration.Nanoseconds() + d2.Duration.Nanoseconds())
}

// ParseDuration returns a Duration struct parsed from a string in format 10h2m3s
func ParseDuration(str string) (Duration, error) {
	d, err := time.ParseDuration(str)
	if err != nil {
		return Duration{0}, err
	}
	return Duration{d}, nil
}

// SubZero returns true if the underlying duration is negative
func (d *Duration) SubZero() bool {
	return d.Duration < 0
}
