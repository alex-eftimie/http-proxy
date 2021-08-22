package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"

	apicontroller "github.com/Alex-Eftimie/api-controller"
)

type con struct {
	MaxThreads     *int
	CurrentThreads int
}

func manageThreads() {
	C.AddHandler("/threads/{serverID:[a-zA-Z0-9]+}", putThreads, "PUT")
	C.AddHandler("/threads/{serverID:[a-zA-Z0-9]+}", getThreads, "GET")
}

func getThreads(w http.ResponseWriter, r *http.Request) {

	ctx := r.Context()
	x, _ := ctx.Value(apicontroller.KeyAuthID).(*APIAuth)

	cr := x.Server.limiter.Current()
	newCon := con{
		MaxThreads:     x.Server.MaxThreads,
		CurrentThreads: cr,
	}

	bytes, _ := json.Marshal(newCon)
	w.Header().Set("Content-Type", "application/json")
	w.Write(bytes)
}
func putThreads(w http.ResponseWriter, r *http.Request) {

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

	newCon := con{}
	err = json.Unmarshal(buf, &newCon)

	if err != nil {
		log.Println(r.RemoteAddr, err)
		w.Header().Set("X-Error", "Invalid MaxThreads Format")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	s.parent.Lock()
	s.Lock()

	s.MaxThreads = newCon.MaxThreads
	s.limiter.SetMax(newCon.MaxThreads)
	newCon.CurrentThreads = s.limiter.Current()

	s.Unlock()
	s.parent.Unlock()

	w.Header().Add("X-Success", "true")
	bytes, _ := json.Marshal(newCon)
	w.Header().Set("Content-Type", "application/json")
	w.Write(bytes)
}
