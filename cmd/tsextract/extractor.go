package main

import (
	"github.com/google/gopacket"
	"github.com/google/gopacket/pcap"
	"os"
)

func extract(inputFileName, outputFileName string) {
	outputFile, err := os.Create(outputFileName)
	if err != nil {
		panic(err)
	}
	defer outputFile.Close()

	handle, err := pcap.OpenOffline(inputFileName)
	if err != nil {
		panic(err)
	}

	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
	for packet := range packetSource.Packets() {
		appLayer := packet.ApplicationLayer()
		if appLayer != nil {
			outputFile.Write(appLayer.Payload())
		}
	}
}
