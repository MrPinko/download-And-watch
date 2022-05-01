package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"strconv"
	"sync"
)

var wg sync.WaitGroup

var link = ""         //direct link to .mp4
var maxGoRoutine = 10 //concurrent download

//watch a mp4 file while downloading it
func main() {

	resp, err := http.Head(link)
	if err != nil {
		log.Fatal(err)
	}
	if resp.StatusCode >= 400 {
		log.Fatal("client status Code : ", resp.StatusCode)
	}
	maps := resp.Header
	length, _ := strconv.Atoi(maps["Content-Length"][0]) // Get the content length from the header request
	limit := length / 5242880                            // number of file part, each of 5 MB
	len_bytes := 5242880                                 // Bytes for each Go-routine

	file, err := os.Create(path.Base(link))
	if err != nil {
		log.Fatal(err)
	}

	if err := file.Truncate(int64(length)); err != nil { //create empty file (all bytes are 0)
		log.Fatal(err)
	}

	//required for streaming (max priority)
	first_last_chunk(len_bytes, file, length)
	wg.Wait()

	middleChunk(limit, len_bytes, file)

	wg.Wait()
	fmt.Println("Download Completed")
}

func first_last_chunk(len_bytes int, file *os.File, length int) {
	wg.Add(2)
	//first chunk
	go downloadPart(0, len_bytes, file)

	//last chunk for (moov in not optimized file)
	go downloadPart(length-len_bytes, length, file)

}

func middleChunk(limit int, len_bytes int, file *os.File) {
	for i := 1; i < limit-1; i++ {
		wg.Add(1)

		min := len_bytes * i       // Min range
		max := len_bytes * (i + 1) // Max range

		go downloadPart(min, max, file)

		if i%maxGoRoutine == 0 {
			wg.Wait()
		}
	}
}

func downloadPart(min int, max int, file *os.File) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", link, nil)
	if err != nil {
		log.Fatal(err)
	}

	range_header := "bytes=" + strconv.Itoa(min) + "-" + strconv.Itoa(max)
	req.Header.Add("Range", range_header)

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	reader, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	file.WriteAt(reader, int64(min)) //replace empty bytes
	wg.Done()
}
