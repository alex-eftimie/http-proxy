package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/shenwei356/util/bytesize"
)

func main() {
	count := 50
	var total int64 = 0
	x := make(chan int64, 100)

	for i := 0; i < count; i++ {

		go func() {
			// file := "http://residential-proxies.reviews/"
			// file := "https://bot.whatismyipaddress.com/"
			file := "https://speed.hetzner.de/100MB.bin"

			d, err := Download(file)

			if err != nil && err != io.EOF {
				fmt.Println(bytesize.ByteSize(d), err)
			}
			x <- d
		}()
	}

	got := 0
	for {
		got++
		sz := <-x
		total += sz
		fmt.Println("Total:", bytesize.ByteSize(total))
		if got >= count {
			break
		}
	}
}

// Download a single file and discard it's contents
func Download(file string) (int64, error) {
	resp, err := http.Get(file)
	if resp != nil {
		log.Println(resp.StatusCode, resp.Status)
	}
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	n, err := io.Copy(ioutil.Discard, resp.Body)
	return n, err
}
