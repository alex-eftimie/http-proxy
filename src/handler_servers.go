package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	apiconfig "github.com/alex-eftimie/api-config"
	apicontroller "github.com/alex-eftimie/api-controller"
	"github.com/alex-eftimie/netutils"
	"github.com/alex-eftimie/utils"
	"github.com/sethvargo/go-password/password"
)

func manageServers() {
	C.AddHandler("/server", putServer, "PUT")
	C.AddHandler("/server", getAllServers, "GET")
	C.AddHandler("/server/{serverID:[a-zA-Z0-9]+}", getServer, "GET")
	C.AddHandler("/server/{serverID:[a-zA-Z0-9]+}", deleteServer, "DELETE")
}

func getAllServers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	x, _ := ctx.Value(apicontroller.KeyAuthID).(*APIAuth)

	if !x.Admin {
		w.Header().Set("X-Error", "Clients are not allowed to do that")
		w.WriteHeader(http.StatusForbidden)
		return
	}
	Ca.Lock()
	bytes, _ := json.Marshal(Ca.Servers)
	Ca.Unlock()

	w.Header().Add("Content-Type", "application/json")
	w.Write(bytes)

}
func getServer(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	x, _ := ctx.Value(apicontroller.KeyAuthID).(*APIAuth)
	x.Server.Lock()
	var bytes []byte

	// if we are admin we copy the whole struct
	bytes, _ = json.Marshal(x.Server)

	// if we are not admin, then we skip sensitive parts
	// like the devices
	if !x.Admin {
		stripped := Server{}
		json.Unmarshal(bytes, &stripped)
		stripped.Devices = nil
		bytes, _ = json.Marshal(&stripped)
	}
	x.Server.Unlock()
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

	u := &netutils.ApiCreateServer{}
	json.Unmarshal(buf, u)
	if u.Email == "" {
		w.Header().Set("X-Error", "Blank Email")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// if username is blank, then convert email to username
	if u.Username == "" {
		u.Username = strings.Replace(u.Email, "@", "+", -1)
	}

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
		token := utils.RandStringBytes(63)
		u.AuthToken = token
	}

	if u.ID == "" {
		serverID := utils.RandStringBytes(15)
		u.ID = serverID
	}

	var bw *PrettyByte = nil

	if u.Bandwidth != nil {
		bw = &PrettyByte{
			Value:    0,
			Readable: *u.Bandwidth,
		}
	}
	var expireAt *utils.Time = nil
	if u.Time != nil {

		expireAt = &utils.Time{Time: time.Now().UTC().Add(u.Time.Duration)}
		u.ExpireAt = expireAt
	}
	Ca.Lock()

	u.Host = Co.ServerIP
	if u.Port == 0 {
		u.Port = Ca.ServerPort
		Ca.ServerPort++
	}
	typ := "UserPass"

	if u.MaxIPs != nil {
		typ = "IP"
		u.Username = ""
		u.Password = ""
	}

	server := Server{
		ID:     u.ID,
		Client: u.Email,
		Addr:   fmt.Sprintf(":%d", u.Port),
		Auth: Auth{
			AuthToken: u.AuthToken,
			Type:      typ,
			User:      u.Username,
			Pass:      u.Password,
		},
		Bytes:      bw,
		MaxThreads: u.MaxThreads,
		MaxIPs:     u.MaxIPs,
		ExpireAt:   expireAt,
		Devices:    u.Devices,
	}
	Ca.Unlock()

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
	apiconfig.Sync(Ca)
	event("ServersChanged")
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

	s.Close()
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
	apiconfig.Sync(Ca)
	event("ServersChanged")
}
