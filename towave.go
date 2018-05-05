package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

var errors chan error
var results chan string
var quit chan bool

func checkErr() {
	for {
		select {
		case err := <-errors:
			fmt.Println(err)
			break
		case res := <-results:
			fmt.Println(res)
			break
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
			continue
		}
		newPath := strings.TrimRight(path, ".smf")
		newPath += ".wav"
		cmd := exec.Command("dd", "if="+path, "of="+newPath, "iflag=binary,count_bytes", "oflag=binary", "bs=1", "skip=16", "status=none")
		_, err := cmd.Output()
		if err != nil {
			errors <- err
			return
		}
		results <- string("wrote " + filepath.Base(newPath))
	}
}

func main() {
	errors = make(chan error)
	results = make(chan string)
	quit = make(chan bool)
	var wg sync.WaitGroup
	threads := 12
	wg.Add(threads)
	s := make(chan string)
	for i := 0; i < threads; i++ {
		go consumer(&wg, s)
	}
	dir := "H:\\extract\\sound"
	go checkErr()
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
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
	for i := 0; i < threads; i++ {
		quit <- true
	}
	wg.Wait()
}
