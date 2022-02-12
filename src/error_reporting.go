package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/davecgh/go-spew/spew"
)

var errorBl chan *string
var errorMp map[string]int

func init() {
	log.Println("Starting Error Reporting Service")
	errorMp = make(map[string]int)
	errorBl = make(chan *string, 10000)
	go processErrors()
	// reportError("Reporting error")
}
func reportError(str string) {
	errorBl <- &str
}

func processErrors() {

	ticker := time.NewTicker(5000 * time.Millisecond)
	for {
		select {
		case <-ticker.C:
			if len(errorMp) == 0 {
				continue
			}
			spew.Dump("Uploading Errors", errorMp)
			uploadErrors(errorMp)
			errorMp = make(map[string]int)
		case errS := <-errorBl:
			v, ok := errorMp[*errS]
			if ok {
				errorMp[*errS] = v + 1
			} else {
				errorMp[*errS] = 1
			}
		}
	}

}

func uploadErrors(mp map[string]int) {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(mp); err != nil {
		// handle error
	}
	req, err := http.NewRequest("PUT", "https://reporting.peertonet.com/api/report", &buf)
	if err != nil {
		// handle error
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != 200 {
		log.Fatalln("Failed to contact error reporting service:", err)
	}
	defer resp.Body.Close()
	x, _ := ioutil.ReadAll(resp.Body)
	log.Println(string(x))
}
