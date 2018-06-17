package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var errors chan string
var results chan string
var quitCheck chan bool
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
			return
		}
	}
}

func smfToWav(oldPath, newPath string) error {
	b, err := ioutil.ReadFile(oldPath)
	if err != nil {
		return err
	}
	b = b[16:]
	ioutil.WriteFile(newPath, b, 0)
	return nil
}

func consumer(wg *sync.WaitGroup, s chan string) {
	defer wg.Done()
	for {
		path, alive := <-s
		if !alive {
			return
		}
		newPath := strings.TrimSuffix(path, ".smf")
		newPath += ".wav"
		err := smfToWav(path, newPath)
		if err != nil {
			errors <- "failed at path " + path
		} else {
			results <- string("wrote " + filepath.Base(newPath))
		}
	}
}

func main() {
	begin := time.Now()
	quitCheck = make(chan bool)
	dir := flag.String("directory", ".", "the directory at which to begin recursively searching")
	threads := flag.Int("threads", 1, "the number of threads with which to work")
	flag.Parse()
	var wg sync.WaitGroup
	wg.Add(*threads)
	s := make(chan string, *threads)
	errors = make(chan string, *threads)
	results = make(chan string, *threads)
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
	close(s)
	wg.Wait()
	quitCheck <- true
	end := time.Now()
	fmt.Println("files processed:", processed)
	if failures > 0 {
		fmt.Println("failures:", failures)
	}
	fmt.Println("time took:", end.Sub(begin))
}
