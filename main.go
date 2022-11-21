package main

import (
	"encoding/csv"
	"fmt"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
	"io"
	"log"
	_ "net/http/pprof"
	"os"
	"sync"
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

Yet another level analyze tool.
`
var Logger = log.New(os.Stdout, "levelman:", log.Lshortfile)

var ReffMap = make(map[string][]string) // referer -> referee
var InvMap = make(map[string]string)    // referee -> referer

func main() {
	fmt.Printf(Art)
	fmt.Printf("Git revision %s\n", Commit)
	fmt.Printf("Built @ %s\n", Time)
	writeMaps("./test_data/in.csv", 0, 1, true)
	directMap := countDirect()
	res := countTotal(directMap)
	Logger.Println(res)
}

func writeMaps(path string, refererColumn int, refereeColumn int, hasHeader bool) {
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

		if hasHeader {
			hasHeader = false
			continue
		}

		if value, ok := ReffMap[record[0]]; ok {
			ReffMap[record[refererColumn]] = append(value, record[refereeColumn])
		} else {
			ReffMap[record[refererColumn]] = []string{record[refereeColumn]}
		}

		InvMap[record[refereeColumn]] = record[refererColumn]

	}
}

func countDirect() map[string]int {
	directMap := make(map[string]int)
	for _, referer := range InvMap {
		if value, ok := directMap[referer]; ok {
			directMap[referer] = value + 1
		} else {
			directMap[referer] = 1
		}
	}
	return directMap
}

func countTotal(directMap map[string]int) map[string]int {
	pipe := make(chan map[string]int)
	done := make(chan map[string]int)
	go countTotalProducer(directMap, pipe)
	go countTotalConsumer(pipe, done)
	return <-done
}

func countTotalProducer(directMap map[string]int, pipe chan<- map[string]int) {
	var wg sync.WaitGroup

	count := func(directMap map[string]int, user string) {
		pipe <- countMemberTotal(directMap, user, []string{})
		wg.Done()
	}

	for u, _ := range ReffMap {
		wg.Add(1)
		go count(directMap, u)
	}
	wg.Wait()
	close(pipe)
}

func countTotalConsumer(pipe <-chan map[string]int, result chan<- map[string]int) {
	totalResult := make(map[string]int)
	for res := range pipe {
		maps.Copy(totalResult, res)
	}
	result <- totalResult
}

func countMemberTotal(directMap map[string]int, user string, used []string) map[string]int {
	if direct, ok := directMap[user]; ok {
		if slices.Contains(used, user) {
			Logger.Fatalln("Loop detected @ user:", user, ", path is", used)
		}
		nextLevel := ReffMap[user]
		used = append(used, user)
		nextLevelCount := 0
		for _, n := range nextLevel {
			nextLevelCount += countMemberTotal(directMap, n, used)[n]
		}
		return map[string]int{user: direct + nextLevelCount}
	} else {
		return map[string]int{user: 0}
	}
}
