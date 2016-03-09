package main

import (
	"os"

	"github.com/google/gopacket"
	"github.com/google/gopacket/pcap"
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
			payload := appLayer.Payload()
			if len(payload) == 1360 {
				outputFile.Write(payload[44:])
			} else {
				outputFile.Write(payload)
			}
		}
	}
}
