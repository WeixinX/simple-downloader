package main

func main() {
	dl := NewDownLoader(
		"https://img2.tapimg.com/bbcode/images/5de1db7e810b34d0598a8ce80f40aea3.jpg?imageView2/2/w/1320/h/9999/q/80/format/jpg/interlace/1/ignore-error/1",
		true,
		"D:\\Goworkspace\\src\\WeixinX\\downloader_learn",
	)
	dl.Run()
}
