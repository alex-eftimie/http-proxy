package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	apicontroller "github.com/Alex-Eftimie/api-controller"
	"github.com/shenwei356/util/bytesize"
)

func manageBandwidth() {
	type U struct {
		Email     string
		Username  string
		Password  string
		Host      string
		Port      int
		Bandwidth string
	}
	C.AddHandler("/bandwidth/{serverID:[a-zA-Z0-9]+}", putBandwidth, "PUT")
	C.AddHandler("/bandwidth/{serverID:[a-zA-Z0-9]+}", getBandwidth, "GET")
}

func getBandwidth(w http.ResponseWriter, r *http.Request) {

	ctx := r.Context()
	x, _ := ctx.Value(apicontroller.KeyAuthID).(*APIAuth)

	bw := x.Server.Bytes

	bytes, _ := json.Marshal(bw)
	w.Header().Set("Content-Type", "application/json")
	w.Write(bytes)
}
func putBandwidth(w http.ResponseWriter, r *http.Request) {

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

	pb := &PrettyByte{}
	json.Unmarshal(buf, pb)

	add := false
	keys, ok := r.URL.Query()["add"]

	disableBW := false
	if ok && keys[0] == "true" {
		add = true
	} else {
		if string(buf) == "{}" {
			disableBW = true
		}
	}

	var nb *PrettyByte = nil
	if disableBW {
		s.parent.Lock()
		s.mux.Lock()
		s.Bytes = nil
		s.mux.Unlock()
		s.parent.Unlock()
	} else {
		extra, err := bytesize.Parse([]byte(pb.Readable))
		if err != nil {
			w.Header().Set("X-Error", "Invalid Byte Format")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		s.parent.Lock()
		s.mux.Lock()

		if add && s.Bytes != nil {
			extra += bytesize.ByteSize(s.Bytes.Value)
		}
		nb = &PrettyByte{
			Value:    int64(extra),
			Readable: extra.String(),
		}
		s.Bytes = nb

		s.SyncBandwidth()
		s.mux.Unlock()
		s.parent.Unlock()

	}
	w.Header().Add("X-Success", "true")
	bytes, _ := json.Marshal(nb)
	w.Header().Set("Content-Type", "application/json")
	w.Write(bytes)
}
