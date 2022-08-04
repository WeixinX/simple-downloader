package main

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"runtime"
	"strings"
	"sync"
)

type DownLoader struct {
	IsConcurrent  bool
	ConcurrentNum int
	Url           *url.URL
	OutDir        string
	FileName      string
}

func NewDownLoader(targetUrl string, isConcurrent bool, outDir string) *DownLoader {
	concurrentNum := runtime.NumCPU()
	if !isConcurrent {
		concurrentNum = 1
	}
	Url, _ := url.Parse(targetUrl)

	return &DownLoader{
		IsConcurrent:  isConcurrent,
		ConcurrentNum: concurrentNum,
		Url:           Url,
		OutDir:        outDir,
		FileName:      path.Base(Url.Path),
	}
}

func (d *DownLoader) Run() {
	fileSize, err := d.getFileSize()
	if err != nil {
		log.Fatalln("get file size error: ", err.Error())
	}

	err = d.multipleDownload(fileSize)
	if err != nil {
		log.Fatalln("multiple file download error: ", err.Error())
	}

	err = d.mergeAll()
	if err != nil {
		log.Fatalln("merge temp file error: ", err.Error())
	}
}

var errRequestFailed = errors.New("request failed")

func (d *DownLoader) getFileSize() (fileSize int64, err error) {
	req, err := http.NewRequest(http.MethodGet, d.Url.String(), nil)
	if err != nil {
		return
	}
	req.Header.Set("Accept-Ranges", "bytes")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	defer func() {
		err = resp.Body.Close()
	}()

	if resp.StatusCode == http.StatusOK && resp.Header.Get("Accept-Ranges") == "bytes" && resp.ContentLength >= 0 {
		return resp.ContentLength, nil
	}

	err = errRequestFailed
	return
}

func (d *DownLoader) multipleDownload(fileSize int64) error {
	wg := sync.WaitGroup{}
	wg.Add(d.ConcurrentNum)

	var (
		step  int64 = fileSize / int64(d.ConcurrentNum)
		start int64 = 0
		end   int64 = start + step
	)

	err := os.MkdirAll(d.OutDir+"/tmp", 0666)
	if err != nil {
		return err
	}

	errChs := make([]chan string, d.ConcurrentNum)
	for i := 0; i < d.ConcurrentNum; i++ {
		if i == d.ConcurrentNum-1 {
			end = fileSize
		}
		errChs[i] = make(chan string)
		go d.partialDownload(i, start, end, &wg, errChs[i])
		start = end + 1
		end = start + step
	}

	wg.Wait()
	// 错误处理
	errStr := strings.Builder{}
	for i, ch := range errChs {
		str := <-ch
		if str == "" {
			continue
		}

		if i == 0 {
			errStr.WriteString("\n")
		}
		errStr.WriteString(fmt.Sprintf("goroutine #%d: %s\n", i, str))
	}
	if errStr.String() != "" {
		return errors.New(errStr.String())
	}

	return nil
}

func (d *DownLoader) partialDownload(id int, start, end int64, wg *sync.WaitGroup, errCh chan<- string) {
	var err error
	defer func() {
		if err != nil {
			errCh <- err.Error()
		} else {
			errCh <- ""
		}
	}()
	defer wg.Done()
	req, err := http.NewRequest(http.MethodGet, d.Url.String(), nil)
	if err != nil {
		return
	}
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", start, end))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	defer func() {
		err = resp.Body.Close()
	}()

	if resp.StatusCode == http.StatusPartialContent {
		var file *os.File
		tmpFileName := fmt.Sprintf("%s/tmp/%s-%d", d.OutDir, d.FileName, id)
		file, err = os.Create(tmpFileName)
		if err != nil {
			return
		}
		defer func() {
			err = file.Close()
		}()
		_, err = io.Copy(file, resp.Body)
		if err != nil {
			return
		}
	}

	log.Printf("#%d: download successed %d-%d\n", id, start, end)
	return
}

func (d *DownLoader) mergeAll() error {
	targetFile, err := os.OpenFile(fmt.Sprintf("%s/%s", d.OutDir, d.FileName),
		os.O_CREATE|os.O_TRUNC|os.O_APPEND, 0666)
	if err != nil {
		return err
	}
	defer func() {
		err = targetFile.Close()
	}()

	for i := 0; i < d.ConcurrentNum; i++ {
		tmpFile, err := os.Open(fmt.Sprintf("%s/tmp/%s-%d", d.OutDir, d.FileName, i))
		if err != nil {
			return err
		}

		content, err := ioutil.ReadAll(tmpFile)
		if err != nil {
			return err
		}
		_, err = targetFile.Write(content)
		err = tmpFile.Close()
	}

	err = os.RemoveAll(fmt.Sprintf("%s/tmp", d.OutDir))
	return err
}
