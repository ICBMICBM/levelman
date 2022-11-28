package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
	"io"
	"log"
	"os"
	"sort"
	"strconv"
	"sync"
	"time"
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

	var refereeFlag = flag.Int("refereeColumn", 1, "下线所在列")
	var refererFlag = flag.Int("refererColumn", 2, "上线所在列")
	var headerFlag = flag.Bool("hasHeader", true, "是否含表头")
	var fileFlag = flag.String("file", "in.csv", "输入文件路径")
	flag.Parse()

	Logger.Println("LevelMan started @", time.Now().Format(time.RFC850))
	writeMaps(*fileFlag, *refererFlag-1, *refereeFlag-1, *headerFlag)
	directMap := countDirect()
	res := countTotal(directMap)

	writeCSV(fmt.Sprintf("./%s_out.csv", *fileFlag), res)
	Logger.Println("Result write to", fmt.Sprintf("%s_out.csv", *fileFlag))
	Logger.Println("LevelMan stopped @", time.Now().Format(time.RFC850))
}

func arrayToString(a []int) []string {
	var newSlice []string
	for _, v := range a {
		newSlice = append(newSlice, strconv.Itoa(v))
	}
	return newSlice
}

func maxIntSlice(v []int) int {
	sort.Ints(v)
	return v[len(v)-1]
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

func countTotal(directMap map[string]int) map[string][]int {
	pipe := make(chan map[string][]int)
	result := make(chan map[string][]int)
	go countTotalProducer(directMap, pipe)
	go countTotalConsumer(pipe, result)
	return <-result
}

func countTotalProducer(directMap map[string]int, pipe chan<- map[string][]int) {
	var wg sync.WaitGroup

	count := func(directMap map[string]int, user string) {
		pipe <- countMemberTotal(directMap, user, []string{}, 0)
		wg.Done()
	}

	for u, _ := range ReffMap {
		wg.Add(1)
		go count(directMap, u)
	}
	wg.Wait()
	close(pipe)
}

func countTotalConsumer(pipe <-chan map[string][]int, result chan<- map[string][]int) {
	totalResult := make(map[string][]int)
	for res := range pipe {
		maps.Copy(totalResult, res)
	}
	result <- totalResult
}

func countMemberTotal(directMap map[string]int, user string, used []string, maxDistance int) map[string][]int {
	if direct, ok := directMap[user]; ok {
		if slices.Contains(used, user) {
			Logger.Fatalln("Loop detected @ user:", user, ", path is", used)
		}
		nextLevel := ReffMap[user]
		used = append(used, user)
		nextLevelCount := 0
		var distances []int

		for _, n := range nextLevel {
			tempResult := countMemberTotal(directMap, n, used, maxDistance)[n]
			nextLevelCount += tempResult[1]
			distances = append(distances, tempResult[2])
		}
		return map[string][]int{user: {direct, direct + nextLevelCount, maxIntSlice(distances) + 1}}
	} else {
		return map[string][]int{user: {direct, 0, 0}}
	}
}

func getNextLevel(user string) []string {
	if nxtLevel, ok := ReffMap[user]; ok {
		return nxtLevel
	} else {
		return []string{}
	}
}

func countMemberTotal2(directMap map[string]int, user string) int32 {
	var total int32
	nxtLevel := []string{user}
	for len(nxtLevel) > 0 {
		nxtLevel = append(nxtLevel, getNextLevel(nxtLevel[0])...)
		slices.Delete(nxtLevel, 0, 0)
		total += 1
	}
	return total
}

func writeCSV(path string, res map[string][]int) {
	csvFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0777)
	if err != nil {
		Logger.Fatalln(err)
	}

	writer := csv.NewWriter(csvFile)
	writer.Write([]string{"user", "direct", "total", "max_distance"})
	for k, v := range res {
		writer.Write(append([]string{k}, arrayToString(v)...))
	}
	writer.Flush()
	csvFile.Close()
}
