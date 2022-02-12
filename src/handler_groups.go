package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"

	apicontroller "github.com/alex-eftimie/api-controller"
)

type group struct {
	DefaultGroup *string
}

func manageGroups() {
	C.AddHandler("/group/{serverID:[a-zA-Z0-9]+}", putGroup, "PUT")
	C.AddHandler("/group/{serverID:[a-zA-Z0-9]+}", getGroup, "GET")
}

func getGroup(w http.ResponseWriter, r *http.Request) {

	ctx := r.Context()
	x, _ := ctx.Value(apicontroller.KeyAuthID).(*APIAuth)

	gr := &group{DefaultGroup: x.Server.DefaultGroup}
	bytes, _ := json.Marshal(gr)
	w.Header().Set("Content-Type", "application/json")
	w.Write(bytes)
}
func putGroup(w http.ResponseWriter, r *http.Request) {

	ctx := r.Context()
	x, _ := ctx.Value(apicontroller.KeyAuthID).(*APIAuth)

	// if x.Admin == false {
	// 	w.Header().Set("X-Error", "Clients are not allowed to do that")
	// 	w.WriteHeader(http.StatusForbidden)
	// 	return
	// }

	s := x.Server

	buf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return
	}

	gr := group{}
	err = json.Unmarshal(buf, &gr)

	if err != nil {
		log.Println(r.RemoteAddr, err)
		w.Header().Set("X-Error", "Invalid Group Format")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	s.parent.Lock()
	s.Lock()

	if _, ok := Co.Proxies[*gr.DefaultGroup]; gr.DefaultGroup != nil && !ok {
		w.Header().Set("X-Error", "Group not found")
		w.WriteHeader(http.StatusBadRequest)
		return

	}

	s.DefaultGroup = gr.DefaultGroup

	// s.MaxThreads = newCon.MaxThreads
	// s.limiter.SetMax(newCon.MaxThreads)
	// newCon.CurrentThreads = s.limiter.Current()

	s.Unlock()
	s.parent.Unlock()

	w.Header().Add("X-Success", "true")
	bytes, _ := json.Marshal(gr)
	w.Header().Set("Content-Type", "application/json")
	w.Write(bytes)
}
