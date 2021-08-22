package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"

	apicontroller "github.com/Alex-Eftimie/api-controller"
)

var userPassRegex = regexp.MustCompile("^[a-z0-9A-Z\\.\\+]+$")

func manageAuth() {
	C.AddHandler("/auth/{serverID:[a-zA-Z0-9]+}", putAuth, "PUT")
	C.AddHandler("/auth/{serverID:[a-zA-Z0-9]+}", getAuth, "GET")
}

func getAuth(w http.ResponseWriter, r *http.Request) {

	ctx := r.Context()
	x, _ := ctx.Value(apicontroller.KeyAuthID).(*APIAuth)

	bw := x.Server.Auth

	bytes, _ := json.Marshal(bw)
	w.Header().Set("Content-Type", "application/json")
	w.Write(bytes)
}
func putAuth(w http.ResponseWriter, r *http.Request) {

	ctx := r.Context()
	x, _ := ctx.Value(apicontroller.KeyAuthID).(*APIAuth)

	s := x.Server

	buf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return
	}

	newAuth := &Auth{}
	err = json.Unmarshal(buf, newAuth)

	if err != nil {
		w.Header().Set("X-Error", "Invalid Auth Format")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if !userPassRegex.Match([]byte(newAuth.User)) ||
		!userPassRegex.Match([]byte(newAuth.Pass)) {
		w.Header().Set("X-Error", "User/Pass valid chars: a-zA-Z0-9.+")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if newAuth.Type != AuthTypeIP {
		newAuth.Type = AuthTypeUserPass
	}

	if s.MaxIPs != nil {
		newAuth.Type = AuthTypeIP
		if len(newAuth.IP) > *s.MaxIPs {
			w.Header().Set("X-Error", fmt.Sprintf("MaxIPs is %d", *s.MaxIPs))
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}

	s.parent.Lock()
	s.Lock()

	if newAuth.AuthToken != s.Auth.AuthToken {
		if !x.Admin {
			newAuth.AuthToken = s.Auth.AuthToken
			w.Header().Set("X-Warning", "Only Admins can change AuthToken")
		} else {
			// remove old token mapping
			Ca.Lock()
			delete(Ca.ServerMap, s.Auth.AuthToken)

			// add new Mapping
			Ca.ServerMap[newAuth.AuthToken] = s
			Ca.Unlock()
		}
	}
	s.Auth = *newAuth

	s.SyncBandwidth()
	s.Unlock()
	s.parent.Unlock()

	w.Header().Add("X-Success", "true")
	bytes, _ := json.Marshal(newAuth)
	w.Header().Set("Content-Type", "application/json")
	w.Write(bytes)
}
