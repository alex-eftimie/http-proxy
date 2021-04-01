package main

// Should be replaced with the new master controller

// func handleServer(w http.ResponseWriter, r *http.Request) {
// 	if r.Method == "PUT" {
// 		var server Server
// 		decoder := json.NewDecoder(r.Body)

// 		if err := decoder.Decode(&server); err != nil {
// 			w.Header().Set("X-Error", "Failed to parse JSON")
// 			w.WriteHeader(http.StatusBadRequest)
// 			return
// 		}
// 		Ca.Mutex.Lock()
// 		defer Ca.Mutex.Unlock()

// 		addr := server.Addr

// 		if server.Auth.Type != "IP" && server.Auth.Type != "UserPass" {
// 			w.Header().Set("X-Error", "Invalid Auth.Type")
// 			w.WriteHeader(http.StatusBadRequest)
// 			return
// 		}

// 		if len(addr) == 0 {
// 			w.Header().Set("X-Error", "Addr is required")
// 			w.WriteHeader(http.StatusBadRequest)
// 			return
// 		}

// 		if _, ok := Ca.ServerMap[addr]; ok {
// 			w.Header().Set("X-Error", "Server exists")
// 			w.WriteHeader(http.StatusBadRequest)
// 			return
// 		}
// 		Ca.Servers = append(Ca.Servers, &server)
// 		RunServer(&server)

// 		w.WriteHeader(http.StatusOK)
// 		bytes, _ := json.MarshalIndent(server, "", "\t")
// 		w.Write(bytes)
// 		return
// 	}

// 	// Method: GET
// 	vars := mux.Vars(r)
// 	srv := vars["server"]

// 	server, ok := Ca.ServerMap[srv]
// 	if !ok {
// 		w.Header().Set("X-Error", "Server Not Found")
// 		w.WriteHeader(http.StatusBadRequest)
// 		return
// 	}
// 	w.WriteHeader(http.StatusOK)
// 	bytes, _ := json.MarshalIndent(server, "", "\t")
// 	w.Write(bytes)
// 	return
// }
// func handleBandwidth(w http.ResponseWriter, r *http.Request) {
// 	vars := mux.Vars(r)
// 	srv := vars["server"]

// 	server, ok := Ca.ServerMap[srv]
// 	if !ok {
// 		w.Header().Set("X-Error", "Server Not Found")
// 		w.WriteHeader(http.StatusBadRequest)
// 		return
// 	}

// 	if r.Method == "GET" {
// 		w.WriteHeader(http.StatusOK)
// 		bytes, _ := json.MarshalIndent(server.Bytes, "", "\t")
// 		w.Write(bytes)
// 		return
// 	}
// 	// Method: PUT

// 	decoder := json.NewDecoder(r.Body)

// 	pb := PrettyByte{}

// 	if err := decoder.Decode(&pb); err != nil {
// 		w.Header().Set("X-Error", "Failed to parse JSON")
// 		w.WriteHeader(http.StatusBadRequest)
// 		return
// 	}

// 	server.mux.Lock()
// 	defer server.mux.Unlock()
// 	server.Bytes = &pb
// 	server.SyncBandwidth()

// 	w.WriteHeader(http.StatusOK)
// 	bytes, _ := json.MarshalIndent(server.Bytes, "", "\t")
// 	w.Write(bytes)
// }
// func handleAuth(w http.ResponseWriter, r *http.Request) {
// 	vars := mux.Vars(r)
// 	srv := vars["server"]

// 	server, ok := Ca.ServerMap[srv]
// 	if !ok {
// 		w.Header().Set("X-Error", "Server Not Found")
// 		w.WriteHeader(http.StatusBadRequest)
// 		return
// 	}

// 	if r.Method == "GET" {
// 		w.WriteHeader(http.StatusOK)
// 		bytes, _ := json.MarshalIndent(server.Auth, "", "\t")
// 		w.Write(bytes)
// 		return
// 	}
// 	// Method: PUT

// 	decoder := json.NewDecoder(r.Body)

// 	auth := Auth{}

// 	if err := decoder.Decode(&auth); err != nil {
// 		w.Header().Set("X-Error", "Failed to parse JSON")
// 		w.WriteHeader(http.StatusBadRequest)
// 		return
// 	}
// 	if auth.Type != "IP" && auth.Type != "UserPass" {
// 		w.Header().Set("X-Error", "Invalid Auth.Type")
// 		w.WriteHeader(http.StatusBadRequest)
// 		return
// 	}

// 	server.mux.Lock()
// 	defer server.mux.Unlock()
// 	server.Auth = auth

// 	w.WriteHeader(http.StatusOK)
// 	bytes, _ := json.MarshalIndent(server.Auth, "", "\t")
// 	w.Write(bytes)

// }
// func RunController() {
// 	r := mux.NewRouter()

// 	r.HandleFunc("/server/{server}", handleServer).Methods("GET", "PUT")
// 	r.HandleFunc("/server/{server}/bandwidth", handleBandwidth).Methods("GET", "PUT")
// 	r.HandleFunc("/server/{server}/auth", handleAuth).Methods("GET", "PUT")

// 	r.Use(authMiddleware)
// 	http.Handle("/", r)

// 	log.Fatal(http.ListenAndServe(":10000", nil))
// }
// func authMiddleware(next http.Handler) http.Handler {
// 	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

// 		if r.Header.Get("X-Token") != Co.AuthToken() {
// 			w.Header().Add("X-Error", "Not authenticated")
// 			http.Error(w, "Forbidden", http.StatusForbidden)
// 			return
// 		}
// 		next.ServeHTTP(w, r)
// 	})
// }
