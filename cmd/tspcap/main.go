package main

import (
	"fmt"
	"log"
	"os"

	"github.com/google/gopacket"
	"github.com/google/gopacket/pcap"
)

const (
	UDPSize  = 1316
	HRTPSize = 1360
)

func main() {
	if len(os.Args) != 2 {
		fmt.Printf("Usage: %s pcap\n", os.Args[0])
		os.Exit(0)
	}

	input, output := os.Args[1], "out.ts"

	handle, err := pcap.OpenOffline(input)
	if err != nil {
		log.Fatal(err)
	}
	defer handle.Close()

	f, err := os.Create(output)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	source := gopacket.NewPacketSource(handle, handle.LinkType())
	for packet := range source.Packets() {
		appLayer := packet.ApplicationLayer()
		if appLayer != nil {
			payload := appLayer.Payload()
			if len(payload) == UDPSize {
				f.Write(payload)
			} else if len(payload) == HRTPSize {
				offset := HRTPSize - UDPSize
				f.Write(payload[offset:HRTPSize])
			}
		}
	}
}
