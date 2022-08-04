package main

import (
	"flag"
	"log"
	"os"

	"github.com/jessevdk/go-flags"
	"github.com/urfave/cli/v2"
)

var (
	Opt Option

	NewOptionParseMap = map[string]newOptionParseFunc{
		"flag":     newFlag,
		"go-flags": newGoFlags,
		"cli":      newCli,
	}
)

type Option struct {
	Url          string `short:"u" long:"url" required:"true" description:"destination download url 目标下载地址"`
	IsConcurrent bool   `short:"c" long:"concurrent" description:"whether to download concurrently 是否并发下载"`
	OutDir       string `short:"o" long:"outpath" default:"./" description:"specify storage folder 指定文件存放目录"`
}

type newOptionParseFunc func()

// flag 库
func newFlag() {
	flag.StringVar(&Opt.Url, "url", "", "destination download url 目标下载地址")
	flag.BoolVar(&Opt.IsConcurrent, "concurrent", false, "whether to download concurrently 是否并发下载")
	flag.StringVar(&Opt.OutDir, "outpath", "./", "specify storage folder 指定文件存放目录")

	flag.Parse()

	NewDownLoader(Opt.Url, Opt.IsConcurrent, Opt.OutDir).Run()
}

// go-flags 库
func newGoFlags() {
	_, err := flags.Parse(&Opt)
	if err != nil {
		log.Fatalln(err)
	}

	NewDownLoader(Opt.Url, Opt.IsConcurrent, Opt.OutDir).Run()
}

// cli 库
func newCli() {
	app := &cli.App{
		Name:  "downloader",
		Usage: "Simple concurrent file downloader",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "url",
				Usage:    "destination download url 目标下载地址",
				Required: true,
				Aliases:  []string{"u"},
			},
			&cli.BoolFlag{
				Name:    "concurrent",
				Usage:   "whether to download concurrently 是否并发下载",
				Value:   false,
				Aliases: []string{"c"},
			},
			&cli.StringFlag{
				Name:    "outpath",
				Usage:   "specify storage folder 指定文件存放目录",
				Value:   "./",
				Aliases: []string{"o"},
			},
		},
		Action: func(c *cli.Context) error {
			Opt.Url = c.String("url")
			Opt.IsConcurrent = c.Bool("concurrent")
			Opt.OutDir = c.String("outpath")

			NewDownLoader(Opt.Url, Opt.IsConcurrent, Opt.OutDir).Run()

			return nil
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatalln(err)
	}
}
