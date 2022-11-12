package mpts

import (
	"bufio"
	"io"
	"os"
	"fmt"
)

const TSPacketSize = 188
const readBufSize = TSPacketSize * 100

func ParseFile(fname string) chan *TsPkt {
	pkts := make(chan *TsPkt)
	go func() {
		f, err := os.Open(fname)
		if err != nil {
			panic(err)
		}
		defer f.Close()

		reader := bufio.NewReaderSize(f, readBufSize)
		var count int64
		for {
			buf := make([]byte, TSPacketSize)
			n, err := io.ReadFull(reader, buf)
			if n == 0 {
				if err != io.EOF {
					panic(err)
				}
				break
			}

			pkt := ParseTsPkt(buf)
			if pkt.SyncByte != 0x47 {
				panic(fmt.Sprintf("Sync byte error at pkt %d", count))
			}

			pkt.Pos = count
			pkts <- pkt
			count += 1
		}
		close(pkts)
	}()
	return pkts
}
