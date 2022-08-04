package main

import "testing"

var dl *DownLoader

func TestMain(m *testing.M) {
	dl = NewDownLoader(
		"https://studygolang.com/dl/golang/go1.19.src.tar.gz",
		true,
		"D:\\Goworkspace\\src\\WeixinX\\downloader_learn",
	)
	m.Run()
}

func TestGetFileSize(t *testing.T) {
	size, err := dl.getFileSize()
	if err != nil {
		t.Error(err)
	}
	t.Log(size, "Bytes")
}

func TestMergeAll(t *testing.T) {
	err := dl.mergeAll()
	if err != nil {
		t.Error()
	}
}

func TestMultipleDownload(t *testing.T) {
	size, err := dl.getFileSize()
	if err != nil {
		t.Error(err)
	}
	dl.setProgressBar(size)
	err = dl.multipleDownload(size)
	if err != nil {
		t.Error(err)
	}
}

func TestRun(t *testing.T) {
	dl.Run()
}
