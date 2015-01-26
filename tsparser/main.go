package main

import (
	"github.com/codegangsta/cli"
	"os"
	"path/filepath"
)

func main() {
	app := cli.NewApp()
	app.Name = "tstool"
	app.Usage = "The tool for transport stream"
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "psi-only",
			Usage: "parse only psi",
		},
		cli.StringFlag{
			Name:  "output",
			Usage: "output directory",
		},
	}
	app.Action = func(c *cli.Context) {
		inputTsFile := c.Args().Get(0)
		if inputTsFile == "" {
			cli.ShowAppHelp(c)
			return
		}
		input := filepath.Base(inputTsFile)
		outdir := c.String("output")
		if outdir == "" {
			outdir = input + ".log"
		}
		psiOnly := c.Bool("psi-only")
		os.Mkdir(outdir, os.ModeDir|0755)
		parse(input, outdir, psiOnly)
	}
	app.Run(os.Args)
}
