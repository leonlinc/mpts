package main

import (
	"github.com/codegangsta/cli"
	"os"
	"path/filepath"
)

func main() {
	app := cli.NewApp()
	app.Name = "tsparser"
	app.Usage = "transport stream parser"
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "psi-only",
			Usage: "parse only psi",
		},
		cli.StringFlag{
			Name:  "output",
			Usage: "output directory",
		},
		cli.IntFlag{
			Name:  "extract",
			Usage: "specify a stream pid to extract",
		},
	}
	app.Action = func(c *cli.Context) {
		inputTsFile := c.Args().Get(0)
		if inputTsFile == "" {
			cli.ShowAppHelp(c)
			return
		}
		inputBaseName := filepath.Base(inputTsFile)
		outdir := c.String("output")
		if outdir == "" {
			outdir = inputBaseName + ".log"
		}
		psiOnly := c.Bool("psi-only")
		os.Mkdir(outdir, os.ModeDir|0755)
		parse(inputTsFile, outdir, psiOnly)
		pidToExtract := c.Int("extract")
		if pidToExtract != 0 {
			extract(inputTsFile, outdir, pidToExtract)
		}
	}
	app.Run(os.Args)
}
