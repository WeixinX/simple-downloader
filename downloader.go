package main

import (
	"bufio"
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

	"github.com/k0kubun/go-ansi"
	"github.com/schollz/progressbar/v3"
)

type DownLoader struct {
	IsConcurrent  bool
	ConcurrentNum int
	Url           *url.URL
	OutDir        string
	FileName      string

	bar *progressbar.ProgressBar
}

func NewDownLoader(targetUrl string, isConcurrent bool, outDir string) *DownLoader {
	concurrentNum := runtime.NumCPU()
	if !isConcurrent {
		concurrentNum = 1
	}
	Url, _ := url.Parse(targetUrl)
	if outDir[len(outDir)-1] == '/' {
		outDir = outDir[:len(outDir)-1]
	}

	return &DownLoader{
		IsConcurrent:  isConcurrent,
		ConcurrentNum: concurrentNum,
		Url:           Url,
		OutDir:        outDir,
		FileName:      path.Base(Url.Path),
	}
}

func (d *DownLoader) setProgressBar(size int64) {
	d.bar = progressbar.NewOptions64(
		size,
		progressbar.OptionSetWriter(ansi.NewAnsiStdout()),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionShowBytes(true),
		progressbar.OptionShowCount(),
		progressbar.OptionSetWidth(30),
		progressbar.OptionSetDescription("[cyan]Downloading...[reset] "),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "[green]=[reset]",
			SaucerHead:    "[green]>[reset]",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}))
}

func (d *DownLoader) Run() {
	fileSize, err := d.getFileSize()
	if err != nil {
		log.Fatalln("get file size error: ", err.Error())
	}

	d.setProgressBar(fileSize)

	err = d.multipleDownload(fileSize)
	if err != nil {
		log.Fatalln("multiple file download error: ", err.Error())
	}

	err = d.mergeAll()
	if err != nil {
		log.Fatalln("merge temp file error: ", err.Error())
	}

	log.Printf("\n===================== [%s] download completed =====================\n", d.FileName)
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

	return 0, errRequestFailed
}

func (d *DownLoader) multipleDownload(fileSize int64) error {
	wg := sync.WaitGroup{}
	wg.Add(d.ConcurrentNum)

	var (
		step  int64 = fileSize / int64(d.ConcurrentNum)
		start int64 = 0
		end   int64 = start + step
	)

	err := os.MkdirAll(d.OutDir+"/tmp", 0777)
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
	// ????????????
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

	if resp.StatusCode == http.StatusPartialContent || resp.StatusCode == http.StatusOK {
		var file *os.File
		tmpFileName := fmt.Sprintf("%s/tmp/%s-%d", d.OutDir, d.FileName, id)
		file, err = os.Create(tmpFileName)
		if err != nil {
			return
		}
		defer func() {
			err = file.Close()
		}()

		_, err = io.Copy(io.MultiWriter(file, d.bar), resp.Body)
		if err != nil {
			return
		}
	}

	log.Printf("#%d: download successed %d-%d\n", id, start, end)
	return
}

func (d *DownLoader) mergeAll() error {
	targetFile, err := os.OpenFile(fmt.Sprintf("%s/%s", d.OutDir, d.FileName),
		os.O_CREATE|os.O_TRUNC|os.O_APPEND|os.O_RDWR, 0777)
	if err != nil {
		return err
	}
	defer func() {
		err = targetFile.Close()
		err = os.RemoveAll(fmt.Sprintf("%s/tmp", d.OutDir))
	}()

	writer := bufio.NewWriter(targetFile)
	for i := 0; i < d.ConcurrentNum; i++ {
		tmpFile, err := os.Open(fmt.Sprintf("%s/tmp/%s-%d", d.OutDir, d.FileName, i))
		if err != nil {
			return err
		}

		content, err := ioutil.ReadAll(tmpFile)
		if err != nil {
			return err
		}
		_, err = writer.Write(content)
		err = tmpFile.Close()
	}

	err = writer.Flush()
	return err
}
