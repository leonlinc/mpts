package main

import (
	"github.com/codegangsta/cli"
	"os"
)

func main() {
	app := cli.NewApp()
	app.Name = "tsextract"
	app.Usage = "Extract ts from pcap"
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "rtp",
			Usage: "decode as rtp",
		},
		cli.StringFlag{
			Name:  "output",
			Usage: "output file",
		},
	}
	app.Action = func(c *cli.Context) {
		inputFileName := c.Args().Get(0)
		if inputFileName == "" {
			cli.ShowAppHelp(c)
			return
		}
		outputFileName := c.String("output")
		if outputFileName == "" {
			outputFileName = "out.ts"
		}
		extract(inputFileName, outputFileName)
	}
	app.Run(os.Args)
}
