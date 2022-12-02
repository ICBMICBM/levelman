package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"github.com/vbauerster/mpb/v8"
	"github.com/vbauerster/mpb/v8/decor"
	"golang.org/x/exp/maps"
	"io"
	"log"
	"os"
	"runtime/pprof"
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
var DirectMap = make(map[string]int32)  // referer -> direct referral count

var usedUserPool = sync.Pool{
	New: func() interface{} { return make([]string, 4096) },
}

func main() {
	fmt.Printf(Art)
	fmt.Printf("Git revision %s\n", Commit)
	fmt.Printf("Built @ %s\n", Time)

	var refereeFlag = flag.Int("refereeColumn", 1, "下线所在列")
	var refererFlag = flag.Int("refererColumn", 2, "上线所在列")
	var headerFlag = flag.Bool("hasHeader", true, "是否含表头")
	var fileFlag = flag.String("file", "in.csv", "输入文件路径")
	Logger.Println("LevelMan started @", time.Now().Format(time.RFC850))

	var cpuProfile = flag.String("cpuprofile", "", "write cpu profile to file")
	var memProfile = flag.String("memprofile", "", "write mem profile to file")
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

	progressBarContainer := mpb.New()
	progressBar := progressBarContainer.New(int64(len(ReffMap)),
		mpb.BarStyle().
			Lbound("╢").
			Filler("▌").
			Tip("▌").
			Padding("░").
			Rbound("╟"),
		mpb.AppendDecorators(decor.Percentage()),
	)

	res := countTotal(progressBar)
	progressBar.Wait()
	writeCSV(fmt.Sprintf("./%s_out.csv", *fileFlag), res)

	Logger.Println("Result write to", fmt.Sprintf("%s_out.csv", *fileFlag))
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

func findIntersect(strs1 []string, strs2 []string) bool {
	intersectMap := map[string]bool{}
	for _, v := range strs1 {
		intersectMap[v] = true
	}
	for _, v := range strs2 {
		if intersectMap[v] {
			intersectMap = nil
			return true
		}
	}
	intersectMap = nil
	return false
}

func countMemberTotal(user string) map[string][]int32 {
	var maxDistance int32
	used := usedUserPool.Get().([]string)
	var total int
	nxtLevel := []string{user}
	for len(nxtLevel) > 0 {
		if findIntersect(used, nxtLevel) {
			Logger.Println("Loop detected @ user:", user)
			return map[string][]int32{user: {DirectMap[user], 0, 0}}
		}
		used = append(used, nxtLevel...)
		nxtLevel = getNextLevel(nxtLevel)
		total += len(nxtLevel)
		maxDistance += 1
	}
	used, nxtLevel = nil, nil
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

func countTotalConsumer(pipe <-chan map[string][]int32, result chan<- map[string][]int32, bar *mpb.Bar) {
	totalResult := make(map[string][]int32)
	for res := range pipe {
		maps.Copy(totalResult, res)
		bar.Increment()
	}
	result <- totalResult
}

func countTotal(bar *mpb.Bar) map[string][]int32 {
	pipe := make(chan map[string][]int32)
	result := make(chan map[string][]int32)
	go countTotalProducer(pipe)
	go countTotalConsumer(pipe, result, bar)
	return <-result
}

func writeCSV(path string, res map[string][]int32) {
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
