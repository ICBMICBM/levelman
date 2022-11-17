package main

import (
	"encoding/csv"
	"fmt"
	"github.com/syndtr/goleveldb/leveldb"
	"io"
	"log"
	"os"
)

var Commit string

func main() {
	fmt.Printf("version %s\n", Commit)
	readReff("./test_data/in.csv")
}

func readReff(path string) {
	// referer -> referee

	csvFile, err := os.Open(path)
	if err != nil {
		log.Fatalln(err)
	}

	r := csv.NewReader(csvFile)
	db, err := leveldb.OpenFile("./test_data/rel1.db", nil)
	if err != nil {
		log.Fatalln(err)
	}
	defer db.Close()

	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			log.Fatalln(err)
		}

		log.Println(record)

	}

}
