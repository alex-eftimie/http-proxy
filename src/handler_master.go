package main

import (
	"errors"
	"net/http"

	apicontroller "github.com/Alex-Eftimie/api-controller"
	"github.com/gorilla/mux"
)

// C Holds the reference to Controller
var C *apicontroller.Controller

// APIAuth holds the server that was authenticated (nil for admins) and Admin is set to true for Master Token
type APIAuth struct {
	Server *Server
	Admin  bool
}

func addHandlers() {
	manageServers()
	manageBandwidth()
	manageTime()
	manageAuth()
	manageThreads()
}

// Run the Controller
func Run() {
	C = apicontroller.NewController()
	C.AuthCallback = func(token string, r *http.Request) (id interface{}, err error) {

		admin := false

		vars := mux.Vars(r)
		sID, ok := vars["serverID"]

		if token == Co.AuthToken() {
			admin = true

			// Admin but no server param is set
			if !ok {
				return &APIAuth{nil, true}, nil
			}

			// if Admin login, search for a server from the url
			token = sID
		}

		if srv := getServerByToken(token); srv != nil {

			// if the token is for a different server, skip
			if srv.ID != sID {
				return nil, errors.New("Invalid Server")
			}
			return &APIAuth{srv, admin}, nil
		}

		return nil, errors.New("Bad Token or Server Not Found")
	}

	addHandlers()

	go C.Run(Co.BindAddr)
}

func getServerByToken(token string) *Server {
	Ca.Lock()
	defer Ca.Unlock()
	return Ca.ServerMap[token]
}
