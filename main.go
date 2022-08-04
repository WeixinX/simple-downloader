package main

func init() {
	f := NewOptionParseMap["cli"]
	f()
}

func main() {
	dl := NewDownLoader(Opt.Url, Opt.IsConcurrent, Opt.OutDir)
	dl.Run()
}
