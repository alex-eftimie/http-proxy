package main

import (
	"github.com/alex-eftimie/netutils"
)

// CheckUser Checks that User, Pass supplied in Proxy-Authorization (ui) matches the server (au)
func CheckUser(ui *netutils.UserInfo, au Auth) bool {
	if ui.User != au.User {
		return false
	}
	if ui.Pass != au.Pass {
		return false
	}
	return true
}
