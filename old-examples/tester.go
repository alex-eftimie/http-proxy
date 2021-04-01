package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
)

func main() {

	myClient := &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(&url.URL{
		Scheme: "http",
		User:   url.UserPassword("alexeftimie", "tTzmG0lgVj5HT7fY"),
		Host:   "127.0.0.1:998",
	})}}

	log.Println("Start http")
	for i := 0; i < 20; i++ {
		res, err := myClient.Get("https://ping.eu")

		fmt.Printf("\r")
		fmt.Printf("%d %d -> %s", i, res.StatusCode, err)
	}
	fmt.Println("\n\n")
	log.Println("Done http")
}
