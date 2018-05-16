package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var errors chan string
var results chan string
var quit chan bool
var quitCheck chan bool
var checkDone chan bool
var processed int
var failures int

func checkErr() {
	for {
		select {
		case err := <-errors:
			fmt.Println(err)
			failures++
			break
		case res := <-results:
			fmt.Println(res)
			processed++
			break
		case <-quitCheck:
			checkDone <- true
			return
		}
	}
}

func waitCheck() {
	quitCheck <- true
	for {
		select {
		case <-checkDone:
			return
		default:
			time.Sleep(time.Millisecond * 100)
		}
	}
}

func consumer(wg *sync.WaitGroup, s chan string) {
	defer wg.Done()
	for {
		var path string
		select {
		case path = <-s:
			break
		case <-quit:
			return
		default:
			time.Sleep(time.Millisecond * 100)
			continue
		}
		newPath := strings.TrimRight(path, ".smf")
		newPath += ".wav"
		cmd := exec.Command("dd", "if="+path, "of="+newPath, "iflag=binary,count_bytes", "oflag=binary", "bs=1", "skip=16", "status=none")
		_, err := cmd.Output()
		if err != nil {
			errors <- "failed at path " + path
		} else {
			results <- string("wrote " + filepath.Base(newPath))
		}
	}
}

func main() {
	begin := time.Now()
	errors = make(chan string)
	results = make(chan string)
	quit = make(chan bool)
	quitCheck = make(chan bool)
	checkDone = make(chan bool)
	dir := flag.String("directory", ".", "the directory at which to begin recursively searching")
	threads := flag.Int("threads", 1, "the number of threads with which to work")
	flag.Parse()
	var wg sync.WaitGroup
	wg.Add(*threads)
	s := make(chan string)
	for i := 0; i < *threads; i++ {
		go consumer(&wg, s)
	}
	go checkErr()
	err := filepath.Walk(*dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Ext(info.Name()) == ".smf" {
			s <- path
		}
		return nil
	})
	if err != nil {
		fmt.Println(err)
	}
	for i := 0; i < *threads; i++ {
		quit <- true
	}
	wg.Wait()
	waitCheck()
	end := time.Now()
	fmt.Println("files processed:", processed)
	if failures > 0 {
		fmt.Println("failures:", failures)
	}
	fmt.Println("time took:", end.Sub(begin))
}
