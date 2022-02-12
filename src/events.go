package main

import (
	"log"
	"os/exec"
	"regexp"
	"strings"

	"github.com/fatih/color"
	"github.com/kballard/go-shellquote"
)

func event(ev string) {
	for r, v := range Co.Events {
		re := regexp.MustCompile(r)
		if re.MatchString(ev) {
			v = strings.Replace(v, "$1", ev, -1)
			params, err := shellquote.Split(v)
			if err != nil {
				color.Red("shellquote.Split error: %s, cmd: %s", err.Error(), v)
			} else {
				// log.Println(params)
				cmd := exec.Command(params[0], params[1:]...)
				out, err := cmd.Output()
				if err != nil {
					log.Println(color.RedString("exec.Command cmd: %s, error: %s, output: %s", v, err.Error(), string(out)))
				} else {
					log.Println(color.GreenString("exec.Command cmd: %s, output: %s", v, string(out)))
				}
			}
		}
	}
}
