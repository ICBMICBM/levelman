package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
)

var Commit string = "Not committed"
var Time string = "Not available"
var Art string = `

.__                     .__                         
|  |   _______  __ ____ |  |   _____ _____    ____  
|  | _/ __ \  \/ // __ \|  |  /     \\__  \  /    \ 
|  |_\  ___/\   /\  ___/|  |_|  Y Y  \/ __ \|   |  \
|____/\___  >\_/  \___  >____/__|_|  (____  /___|  /
          \/          \/           \/     \/     \/
`
var Logger = log.New(os.Stdout, "levelman:", log.Lshortfile)

var ReffMap = make(map[string][]string)   // referer -> referee
var InversedMap = make(map[string]string) // referee -> referer

func main() {
	fmt.Printf(Art)
	fmt.Printf("Version %s\n", Commit)
	fmt.Printf("Built @ %s\n", Time)
	readReff("./test_data/in.csv")
}

func readReff(path string) {
	// referer -> referee

	csvFile, err := os.Open(path)
	if err != nil {
		Logger.Fatalln(err)
	}
	r := csv.NewReader(csvFile)

	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			Logger.Fatalln(err)
		}

		if value, ok := ReffMap[record[0]]; ok {
			ReffMap[record[0]] = append(value, record[1])
		} else {
			ReffMap[record[0]] = []string{record[1]}
		}

		if err != nil {
			Logger.Fatalln(err)
		}
	}
}
