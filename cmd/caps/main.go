package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"os"

	"github.com/google/gopacket"
	"github.com/google/gopacket/pcap"
	"github.com/leonlinc/mpts"
)

func handlePayload(b []byte, t int64) {
	for i := 0; i < 7; i++ {
		offset := 188 * i
		if b[offset] != 0x47 {
			fmt.Println("Sync Byte Error")
		}
		pkt := mpts.ParseTsPkt(b[offset:])
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

	input, output := os.Args[1], os.Args[1]+".ts"

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
			if len(payload) != 1316 {
				// RTP header minimum size 12 byte
				offset := 12
				// Ignore CSRC for now
				offset += 0
				// Extension header
				offset += 4 + 4*int(binary.BigEndian.Uint16(payload[offset+2:]))
				payload = payload[offset:]
			}
			// For debugging real-time PCR jitter
			// currTime := packet.Metadata().CaptureInfo.Timestamp.UnixNano() / 1000000
			// handlePayload(payload, currTime)
			f.Write(payload)
		}
	}
}
