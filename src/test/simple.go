package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
)

func main() {

	uri := "https://bot.whatismyipaddress.com/"

	proxyURL, err := url.Parse("http://alexandru+alexeftimie.ro:fV0Fu9pa84Mi5xI3@127.0.0.1:1006")
	myClient := &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)}}

	resp, err := myClient.Get(uri)
	if err != nil {
		log.Fatalln(err)
	}

	if resp != nil {
		log.Println(resp.StatusCode, resp.Status)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}

	log.Println(string(body))
}
