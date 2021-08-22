package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"

	apicontroller "github.com/Alex-Eftimie/api-controller"
)

func manageTime() {
	type U struct {
		Email     string
		Username  string
		Password  string
		Host      string
		Port      int
		Bandwidth string
	}
	C.AddHandler("/time/{serverID:[a-zA-Z0-9]+}", putTime, "PUT")
	C.AddHandler("/time/{serverID:[a-zA-Z0-9]+}", getTime, "GET")
}

func getTime(w http.ResponseWriter, r *http.Request) {

	ctx := r.Context()
	x, _ := ctx.Value(apicontroller.KeyAuthID).(*APIAuth)

	bw := x.Server.Bytes

	bytes, _ := json.Marshal(bw)
	w.Header().Set("Content-Type", "application/json")
	w.Write(bytes)
}
func putTime(w http.ResponseWriter, r *http.Request) {

	ctx := r.Context()
	x, _ := ctx.Value(apicontroller.KeyAuthID).(*APIAuth)

	if x.Admin == false {
		w.Header().Set("X-Error", "Clients are not allowed to do that")
		w.WriteHeader(http.StatusForbidden)
		return
	}

	s := x.Server

	buf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return
	}

	type resp struct {
		ExpireAt *PTime
	}
	type dur struct {
		Time *Duration
	}

	d := &dur{}
	json.Unmarshal(buf, d)

	add := false
	keys, ok := r.URL.Query()["add"]

	disableTime := false
	if ok && keys[0] == "true" {
		add = true
	} else {
		if string(buf) == "{}" {
			disableTime = true
		}
	}

	var nb *resp = nil
	if disableTime {
		s.parent.Lock()
		s.Lock()
		s.ExpireAt = nil
		s.Unlock()
		s.parent.Unlock()
	} else {
		s.parent.Lock()
		s.Lock()

		if add && s.ExpireAt != nil {
			s.ExpireAt = &PTime{s.ExpireAt.Add(d.Time.Duration)}
		} else {
			s.ExpireAt = &PTime{time.Now().Add(d.Time.Duration)}
		}
		nb = &resp{
			ExpireAt: s.ExpireAt,
		}

		s.SyncBandwidth()
		s.Unlock()
		s.parent.Unlock()

	}
	w.Header().Add("X-Success", "true")
	bytes, _ := json.Marshal(nb)
	w.Header().Set("Content-Type", "application/json")
	w.Write(bytes)
}
