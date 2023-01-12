package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"golang.org/x/exp/maps"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime/pprof"
	"strconv"
	"strings"
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
var DirectMap = make(map[string]int32)  // referer -> direct referral count

var usedUserPool = sync.Pool{
	New: func() interface{} { return new(noDup) },
}

type noDup struct {
	kv map[string]bool
}

func (n *noDup) Test(s []string) bool {
	for _, v := range s {
		if n.kv[v] {
			return true
		}
	}
	return false
}

func (n *noDup) Add(s []string) {
	for _, v := range s {
		n.kv[v] = true
	}
}

func (n *noDup) Init() {
	n.kv = make(map[string]bool)
}

func main() {
	fmt.Printf(Art)
	fmt.Printf("Git revision %s\n", Commit)
	fmt.Printf("Built @ %s\n", Time)

	var refereeFlag = flag.Int("ee", 1, "下线所在列")
	var refererFlag = flag.Int("er", 2, "上线所在列")
	var headerFlag = flag.Bool("h", true, "是否含表头")
	var fileFlag = flag.String("f", "in.csv", "输入文件路径")
	Logger.Println("LevelMan started @", time.Now().Format(time.RFC850))

	var cpuProfile = flag.String("cpu", "", "write cpu profile to file")
	var memProfile = flag.String("mem", "", "write mem profile to file")
	flag.Parse()

	if *cpuProfile != "" {
		f, err := os.Create(*cpuProfile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	if *memProfile != "" {
		f, err := os.Create(*memProfile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.WriteHeapProfile(f)
		f.Close()
	}

	writeMaps(*fileFlag, *refererFlag-1, *refereeFlag-1, *headerFlag)
	DirectMap = countDirect()

	res := countTotal()
	writeCSV(*fileFlag, res)

	Logger.Println("LevelMan stopped @", time.Now().Format(time.RFC850))
}

func arrayToString(a []int32) []string {
	var newSlice []string
	for _, v := range a {
		newSlice = append(newSlice, strconv.Itoa(int(v)))
	}
	return newSlice
}

func writeMaps(path string, refererColumn int, refereeColumn int, hasHeader bool) {
	absPath, err := filepath.Abs(path)
	Logger.Println("absPath is", absPath)
	csvFile, err := os.Open(absPath)
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

		if value, ok := ReffMap[record[refererColumn]]; ok {
			ReffMap[record[refererColumn]] = append(value, record[refereeColumn])
		} else {
			ReffMap[record[refererColumn]] = []string{record[refereeColumn]}
		}

		InvMap[record[refereeColumn]] = record[refererColumn]

	}
}

func countDirect() map[string]int32 {
	directMap := make(map[string]int32)
	for _, referer := range InvMap {
		if value, ok := directMap[referer]; ok {
			directMap[referer] = value + 1
		} else {
			directMap[referer] = 1
		}
	}
	return directMap
}

func getNextLevel(user []string) []string {
	var nxtLevel []string
	for _, u := range user {
		if nxt, ok := ReffMap[u]; ok {
			nxtLevel = append(nxtLevel, nxt...)
		} else {
			continue
		}
	}
	return nxtLevel
}

func countMemberTotal(user string) map[string][]int32 {
	var maxDistance int32
	used := usedUserPool.Get().(*noDup)
	used.Init()
	var total int
	nxtLevel := []string{user}
	for len(nxtLevel) > 0 {
		if used.Test(nxtLevel) {
			Logger.Println("Loop detected @ user:", user)
			return map[string][]int32{user: {DirectMap[user], 0, 0}}
		}
		used.Add(nxtLevel)
		nxtLevel = getNextLevel(nxtLevel)
		total += len(nxtLevel)
		maxDistance += 1
	}
	nxtLevel = nil
	usedUserPool.Put(used)
	return map[string][]int32{user: {DirectMap[user], int32(total), maxDistance - 1}}
}

func countTotalProducer(pipe chan<- map[string][]int32) {
	var wg sync.WaitGroup

	count := func(user string) {
		pipe <- countMemberTotal(user)
		wg.Done()
	}

	for u, _ := range ReffMap {
		wg.Add(1)
		go count(u)
	}
	wg.Wait()
	close(pipe)
}

func countTotalConsumer(pipe <-chan map[string][]int32, result chan<- map[string][]int32) {
	totalResult := make(map[string][]int32)
	for res := range pipe {
		maps.Copy(totalResult, res)
	}
	result <- totalResult
}

func countTotal() map[string][]int32 {
	pipe := make(chan map[string][]int32)
	result := make(chan map[string][]int32)
	go countTotalProducer(pipe)
	go countTotalConsumer(pipe, result)
	return <-result
}

func writeCSV(path string, res map[string][]int32) {
	dir, fileName, ext := filepath.Dir(path), filepath.Base(path), filepath.Ext(path)
	csvFile, err := os.OpenFile(filepath.Join(dir, strings.TrimSuffix(fileName, ext)+"_out.csv"), os.O_WRONLY|os.O_CREATE, 0777)
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
	Logger.Println("Writing result to", csvFile.Name())
}
