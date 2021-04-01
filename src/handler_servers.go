package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	apicontroller "github.com/Alex-Eftimie/api-controller"
	"github.com/sethvargo/go-password/password"
)

type apiCreateServer struct {
	AuthToken string
	Email     string
	Username  string
	Password  string
	Host      string
	Port      int
	ID        string
	Bandwidth *string
}

func manageServers() {
	C.AddHandler("/server", putServer, "PUT")
	C.AddHandler("/server/{serverID:[a-zA-Z0-9]+}", getServer, "GET")
	C.AddHandler("/server/{serverID:[a-zA-Z0-9]+}", deleteServer, "DELETE")
}

func getServer(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	x, _ := ctx.Value(apicontroller.KeyAuthID).(*APIAuth)
	x.Server.mux.Lock()
	bytes, _ := json.Marshal(x.Server)
	x.Server.mux.Unlock()
	w.Header().Add("Content-Type", "application/json")
	w.Write(bytes)
}
func putServer(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	x, _ := ctx.Value(apicontroller.KeyAuthID).(*APIAuth)

	if !x.Admin {
		w.Header().Set("X-Error", "Clients are not allowed to do that")
		w.WriteHeader(http.StatusForbidden)
		return
	}

	buf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.Header().Set("X-Error", "Error reading request body")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	u := &apiCreateServer{}
	json.Unmarshal(buf, u)
	if u.Email == "" {
		w.Header().Set("X-Error", "Blank Email")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	u.Username = strings.Replace(u.Email, "@", "+", -1)
	if u.Password == "" {
		res, err := password.Generate(16, 6, 0, false, false)
		if err != nil {
			w.Header().Set("X-Error", "Could not generate password")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		u.Password = res
	}

	if u.AuthToken == "" {
		token := RandStringBytes(63)
		u.AuthToken = token
	}

	if u.ID == "" {
		serverID := RandStringBytes(15)
		u.ID = serverID
	}

	var bw *PrettyByte = nil

	if u.Bandwidth != nil {
		bw = &PrettyByte{
			Value:    0,
			Readable: *u.Bandwidth,
		}
	}

	Ca.Mutex.Lock()

	u.Host = Co.ServerIP
	u.Port = Ca.ServerPort
	Ca.ServerPort++

	server := Server{
		ID:     u.ID,
		Client: u.Email,
		Addr:   fmt.Sprintf(":%d", u.Port),
		Auth: Auth{
			AuthToken: u.AuthToken,
			Type:      "UserPass",
			User:      u.Username,
			Pass:      u.Password,
		},
		Bytes: bw,
	}
	Ca.Mutex.Unlock()

	if err := RunServer(&server); err == nil {
		Ca.Lock()
		Ca.Servers = append(Ca.Servers, &server)
		Ca.Unlock()
	} else {
		w.Header().Set("X-Error", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	bytes, _ := json.Marshal(u)
	w.Header().Add("Content-Type", "application/json")
	w.Write(bytes)
}

func deleteServer(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	x, _ := ctx.Value(apicontroller.KeyAuthID).(*APIAuth)

	if !x.Admin {
		w.Header().Set("X-Error", "Clients are not allowed to do that")
		w.WriteHeader(http.StatusForbidden)
		return
	}

	s := x.Server
	log.Println("Deleting server", s.ID, s.Addr)

	s.Server.Close()
	Ca.Lock()

	// delete from
	for i := 0; i < len(Ca.Servers); i++ {
		if Ca.Servers[i] != s {
			continue
		}

		// Remove element at i
		Ca.Servers = append(Ca.Servers[0:i], Ca.Servers[i+1:]...)

		break
	}

	// remove mappings
	delete(Ca.ServerMap, s.ID)
	delete(Ca.ServerMap, s.Auth.AuthToken)
	delete(Ca.ServerMap, s.Addr)

	Ca.Unlock()
}
