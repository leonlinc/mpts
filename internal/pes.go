package mpts

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type PesRecord struct {
	BaseRecord
	curpkt *PesPkt
	Pkts   []*PesPkt
}

func (s *PesRecord) Process(pkt *TsPkt) {
	s.LogAdaptFieldPrivData(pkt)
	if pkt.PUSI == 1 {
		if s.curpkt != nil {
			s.Pkts = append(s.Pkts, s.curpkt)
		}
		s.curpkt = &PesPkt{}
		s.curpkt.Pos = pkt.Pos
		s.curpkt.Pcr = s.BaseRecord.PcrTime
		s.curpkt.PcrPos = s.BaseRecord.PcrPos
		var startcode = []byte{0, 0, 1}
		if 0 == bytes.Compare(startcode, pkt.Data[0:3]) {
			hlen := s.curpkt.Read(pkt.Data)
			if s.curpkt.StreamId == 0xBE {
				s.curpkt = nil
			} else {
				pkt.Data = pkt.Data[hlen:]
			}
		}
	}
	if s.curpkt != nil {
		s.curpkt.Size += int64(len(pkt.Data))
	}
}

func (s *PesRecord) Flush() {
	if s.curpkt != nil {
		s.Pkts = append(s.Pkts, s.curpkt)
	}
}

func (s *PesRecord) Report(root string) {
	fname := filepath.Join(root, strconv.Itoa(s.Pid)+".csv")
	w, err := os.Create(fname)
	if err != nil {
		panic(err)
	}
	defer w.Close()

	header := "Pos, Size, PCR, PcrPos, PTS, DTS, (DTS-PCR)"
	fmt.Fprintln(w, header)
	for _, p := range s.Pkts {
		pcr := p.Pcr / 300
		dts := p.Dts
		if dts == 0 {
			dts = p.Pts
		}
		cols := []string{
			strconv.FormatInt(p.Pos, 10),
			strconv.FormatInt(p.Size, 10),
			strconv.FormatInt(pcr, 10),
			strconv.FormatInt(p.PcrPos, 10),
			strconv.FormatInt(p.Pts, 10),
			strconv.FormatInt(dts, 10),
			strconv.FormatInt(dts-pcr, 10),
		}
		fmt.Fprintln(w, strings.Join(cols, ", "))
	}
}
