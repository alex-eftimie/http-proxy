package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	apicontroller "github.com/alex-eftimie/api-controller"
	"github.com/alex-eftimie/utils"
)

func manageDevices() {
	C.AddHandler("/devices/{serverID:[a-zA-Z0-9]+}", putDevices, "PUT")
	C.AddHandler("/devices/{serverID:[a-zA-Z0-9]+}", getDevices, "GET")
}

func getDevices(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	x, _ := ctx.Value(apicontroller.KeyAuthID).(*APIAuth)

	if !x.Admin {
		w.Header().Set("X-Error", "Clients are not allowed to do that")
		w.WriteHeader(http.StatusForbidden)
		return
	}

	bw := x.Server.Devices

	bytes, _ := json.Marshal(bw)
	w.Header().Set("Content-Type", "application/json")
	w.Write(bytes)
}
func putDevices(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	x, _ := ctx.Value(apicontroller.KeyAuthID).(*APIAuth)

	if !x.Admin {
		w.Header().Set("X-Error", "Clients are not allowed to do that")
		w.WriteHeader(http.StatusForbidden)
		return
	}
	s := x.Server

	buf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return
	}

	newDevices := make(map[string]string)
	err = json.Unmarshal(buf, &newDevices)

	if err != nil {
		utils.Debugf(999, "putDevices error: %s", err.Error())
		w.Header().Set("X-Error", "Invalid Devices Format")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	s.parent.Lock()
	s.Lock()

	s.Devices = newDevices

	s.SyncBandwidth()
	s.Unlock()
	s.parent.Unlock()

	w.Header().Add("X-Success", "true")
	bytes, _ := json.Marshal(newDevices)
	w.Header().Set("Content-Type", "application/json")
	w.Write(bytes)
}
