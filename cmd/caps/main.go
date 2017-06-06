package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/google/gopacket"
	"github.com/google/gopacket/pcap"
	"github.com/leonlinc/mpts"
)

const TsPktSize = 188

var PrintRecords bool

func init() {
	flag.BoolVar(&PrintRecords, "p", false, "print timing records")
}

type PcrRecord struct {
	index   int
	pcrTime int64
}

type MuxRecord struct {
	PcrRecord
	muxerTime uint32
}

func main() {
	flag.Parse()

	if len(flag.Args()) != 1 {
		fmt.Printf("Usage: caps [options] <pcap-file>\n")
		os.Exit(1)
	}

	input := flag.Args()[0]

	h, err := pcap.OpenOffline(input)
	if err != nil {
		log.Fatal(err)
	}
	defer h.Close()

	f, err := os.Create(input + ".ts")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	s := gopacket.NewPacketSource(h, h.LinkType())
	process(s, f)
}

func process(source *gopacket.PacketSource, output *os.File) {
	index := 0
	for packet := range source.Packets() {
		capTime := packet.Metadata().CaptureInfo.Timestamp.UnixNano() / 1e6
		payload, records := parsePacket(packet)
		if payload != nil {
			output.Write(payload)
		}
		if PrintRecords {
			for _, record := range records {
				fmt.Println(capTime, index+record.index, record.pcrTime, record.muxerTime)
			}
		}
		index += len(payload) / TsPktSize
	}
}

func parsePacket(packet gopacket.Packet) (data []byte, muxRecords []MuxRecord) {
	appLayer := packet.ApplicationLayer()
	if appLayer == nil {
		fmt.Fprintln(os.Stderr, "error: no application payload")
		return
	}

	payload := appLayer.Payload()

	var muxerTimes []uint32
	if len(payload) != 1316 {
		data, muxerTimes = parseHrtp(payload)
	} else {
		data = payload
	}

	pcrRecords := parseTsData(data)
	for _, pcrRecord := range pcrRecords {
		var muxerTime uint32
		if len(muxerTimes) > pcrRecord.index {
			muxerTime = muxerTimes[pcrRecord.index]
		}
		muxRecords = append(muxRecords, MuxRecord{pcrRecord, muxerTime})
	}
	return
}

func parseHrtp(data []byte) (payload []byte, muxerTimes []uint32) {
	// RTP header has a minimum size of 12 bytes.
	offset := 12
	extension := (data[0] >> 4) & 1
	// TODO: assume no CSRC
	if extension == 1 {
		// Extension
		extensionLength := binary.BigEndian.Uint16(data[offset+2:])
		// Extension header
		offset += 4
		for i := 0; i < int(extensionLength); i++ {
			muxerTime := binary.BigEndian.Uint32(data[offset+4*i:])
			muxerTimes = append(muxerTimes, muxerTime)
		}
		// Extension entries
		offset += 4 * int(extensionLength)
	}
	payload = data[offset:]
	return
}

func parseTsData(data []byte) []PcrRecord {
	var pcrRecords []PcrRecord
	cnt := len(data) / TsPktSize
	for i := 0; i < cnt; i++ {
		offset := i * TsPktSize
		if data[offset] != 0x47 {
			fmt.Fprintln(os.Stderr, "error: TS sync byte != 0x47")
			continue
		}
		pkt := mpts.ParseTsPkt(data[offset:])
		if pcr, ok := pkt.PCR(); ok {
			pcrRecords = append(pcrRecords, PcrRecord{i, pcr})
		}
	}
	return pcrRecords
}
