package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/pcap"
	"github.com/leonlinc/mpts"
)

var pcrFlag = flag.Bool("pcr", false, "a bool")

func main() {
	flag.Parse()

	if len(flag.Args()) != 1 {
		fmt.Printf("Usage: caps [options] <pcap-file>\n")
		os.Exit(1)
	}

	input := flag.Args()[0]
	handle, err := pcap.OpenOffline(input)
	if err != nil {
		log.Fatal(err)
	}
	defer handle.Close()

	f, err := os.Create(input + ".ts")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	source := gopacket.NewPacketSource(handle, handle.LinkType())
	for packet := range source.Packets() {
		payload := extract(packet)
		if payload != nil {
			f.Write(payload)
		}
	}
}

func extract(packet gopacket.Packet) (payload []byte) {
	appLayer := packet.ApplicationLayer()
	if appLayer != nil {
		payload = appLayer.Payload()
		if len(payload) != 1316 {
			// RTP header has a minimum size of 12 bytes.
			offset := 12
			extension := (payload[0] >> 4) & 1
			// TODO: assume no CSRC
			if extension == 1 {
				// Extension
				extensionLength := binary.BigEndian.Uint16(payload[offset+2:])
				// Extension header
				offset += 4
				// Extension entries
				offset += 4 * int(extensionLength)
			}
			payload = payload[offset:]
		}
		if *pcrFlag {
			timestamp := packet.Metadata().CaptureInfo.Timestamp.UnixNano() / int64(time.Millisecond)
			timing(payload, timestamp)
		}
	}
	return
}

func timing(payload []byte, timestamp int64) {
	for i := 0; i < 7; i++ {
		offset := 188 * i
		if payload[offset] != 0x47 {
			fmt.Println("Sync Byte Error")
		} else {
			pkt := mpts.ParseTsPkt(payload[offset:])
			if pcr, ok := pkt.PCR(); ok {
				fmt.Println(timestamp, pcr/27000)
			}
		}
	}
}
