package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/shenwei356/util/bytesize"
)

func count() {
	var columns []int
	err := json.Unmarshal([]byte(cols), &columns)
	if err != nil {
		log.Fatalln("Cols error:", err)
	}
	var total int64 = 0
	for _, fileName := range flag.Args() {
		total += readFile(fileName, columns)
	}

	readable := bytesize.ByteSize(total)

	log.Println("Total Bytes:", total, readable)
}

func readFile(fileName string, cols []int) int64 {

	rex := regexp.MustCompile(`[^0-9\.\-]`)
	var sum int64 = 0
	file, err := os.Open(fileName)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	// optionally, resize scanner's capacity for lines over 64K, see next example
	for scanner.Scan() {
		text := scanner.Text()
		columns := strings.Split(text, sep)
		for _, c := range cols {

			// skip lines with too few columns
			if len(columns) <= c {
				continue
			}
			cleanString := rex.ReplaceAllString(columns[c], "")
			intVal, _ := strconv.ParseInt(cleanString, 10, 64)
			sum += intVal
			columns[c] = color.RedString(columns[c])
		}
		readable := bytesize.ByteSize(sum)
		if prnt {
			fmt.Println(readable, "##", strings.Join(columns, sep))
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
	return sum
}
