package main

import (
	"encoding/json"
	"log"
	"time"

	"github.com/davecgh/go-spew/spew"
)

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

func main() {
	x := &PTime{time.Now()}

	x = &PTime{x.AddDate(0, 0, 10)}

	d, err := json.Marshal(x)
	spew.Dump(d, err)

	log.Println("M:", string(d))

	var pt PTime
	err = json.Unmarshal(d, &pt)

	spew.Dump("mpt", pt, err)
	log.Println(pt)
}
