package main

import (
	"encoding/base64"
	"net/http"
	"strings"
)

// UserInfo is a simple User, Pass struct
type UserInfo struct {
	User string
	Pass string
}

// GetAuth decodes the Proxy-Authorization header
// @param *http.Request
// @returns *UserInfo
func GetAuth(r *http.Request) *UserInfo {
	s := r.Header.Get("Proxy-Authorization")
	if s == "" {
		return nil
	}
	ss := strings.Split(s, " ")
	if ss[0] != "Basic" {
		return nil
	}
	b, err := base64.StdEncoding.DecodeString(ss[1])
	if err != nil {
		return nil
	}
	ss = strings.Split(string(b), ":")
	return &UserInfo{
		User: ss[0],
		Pass: ss[1],
	}
}

// CheckUser Checks that User, Pass supplied in Proxy-Authorization (ui) matches the server (au)
func CheckUser(ui *UserInfo, au Auth) bool {
	if ui.User != au.User {
		return false
	}
	if ui.Pass != au.Pass {
		return false
	}
	return true
}
