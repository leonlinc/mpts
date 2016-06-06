package main

import (
	"fmt"
	"log"
	"os"

	"github.com/google/gopacket"
	"github.com/google/gopacket/pcap"
	"github.com/leonlinc/ts"
)

const (
	UDPSize  = 1316
	HRTPSize = 1360
)

func handlePayload(b []byte, t int64) {
	for i := 0; i < 7; i++ {
		offset := 188 * i
		if b[offset] != 0x47 {
			fmt.Println("Sync Byte Error")
		}
		pkt := ts.ParseTsPkt(b)
		if pcr, ok := pkt.PCR(); ok {
			fmt.Println(t, pcr/27000)
		}
	}
}

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
		currTime := packet.Metadata().CaptureInfo.Timestamp.UnixNano() / 1000000
		appLayer := packet.ApplicationLayer()
		if appLayer != nil {
			payload := appLayer.Payload()
			if len(payload) == UDPSize {
				handlePayload(payload, currTime)
				f.Write(payload)
			} else if len(payload) == HRTPSize {
				offset := HRTPSize - UDPSize
				handlePayload(payload[offset:HRTPSize], currTime)
				f.Write(payload[offset:HRTPSize])
			}
		}
	}
}
