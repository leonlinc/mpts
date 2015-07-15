package ts

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

var NalUnitType []string = []string{
	"unspecified",
	"slice_non_idr",
	"slice_partition_A",
	"slice_partition_B",
	"slice_partition_C",
	"slice_idr",
	"sei",
	"seq_param_set",
	"pic_param_set",
	"au_delimiter",
	"end_of_seq",
	"end_of_stream",
	"filler_data",
	"seq_param_set_ext",
	"prefix",
	"sub_seq_param_set",
}

func GetNalUnitType(b int) string {
	b = b & 0x1F
	if b < len(NalUnitType) {
		return NalUnitType[b]
	} else {
		return NalUnitType[0]
	}
}

func ParseNalUnits(data []byte) []string {
	var nals []string
	var pos int
	var startcode = []byte{0, 0, 1}
	var startlen = len(startcode)
	for pos+5 < len(data) {
		if bytes.Compare(startcode, data[pos:pos+startlen]) == 0 {
			pos += startlen
			nal := GetNalUnitType(int(data[pos]))
			nals = append(nals, nal)
		}
		pos += 1
	}
	return nals
}

type H264Record struct {
	BaseRecord
	Root      string
	Pid       int
	curpkt    *PesPkt
	Pkts      []*PesPkt
	Nals      [][]string
	IFrameLog *os.File
}

func (r *H264Record) LogIFrame(i IFrameInfo) {
	if r.IFrameLog == nil {
		var pid string = strconv.Itoa(r.Pid)
		var err error
		fname := filepath.Join(r.Root, pid+"-iframe"+".csv")
		r.IFrameLog, err = os.Create(fname)
		if err != nil {
			panic(err)
		}
		header := "Pos, PTS, Key"
		fmt.Fprintln(r.IFrameLog, header)
	}
	cols := []string{
		strconv.FormatInt(i.Pos, 10),
		strconv.FormatInt(i.Pts, 10),
		strconv.FormatBool(i.Key),
	}
	fmt.Fprintln(r.IFrameLog, strings.Join(cols, ", "))
}

func (s *H264Record) Process(pkt *TsPkt) {
	if pkt.PUSI == 1 {
		if s.curpkt != nil {
			nals := ParseNalUnits(s.curpkt.Data)
			for _, nal := range nals {
				if nal == "slice_idr" {
					info := IFrameInfo{}
					info.Pos = s.curpkt.Pos
					info.Pts = s.curpkt.Pts
					info.Key = true
					s.LogIFrame(info)
				}
			}
			s.Nals = append(s.Nals, nals)
			s.Pkts = append(s.Pkts, s.curpkt)
		}
		s.curpkt = &PesPkt{}
		s.curpkt.Pos = pkt.Pos
		s.curpkt.Pcr = s.BaseRecord.PcrTime
		var startcode = []byte{0, 0, 1}
		if 0 == bytes.Compare(startcode, pkt.Data[0:3]) {
			hlen := s.curpkt.Read(pkt)
			pkt.Data = pkt.Data[hlen:]
		}
	}
	if s.curpkt != nil {
		s.curpkt.Size += int64(len(pkt.Data))
		s.curpkt.Data = append(s.curpkt.Data, pkt.Data...)
	}
}

func (s *H264Record) Flush() {
	if s.curpkt != nil {
		nals := ParseNalUnits(s.curpkt.Data)
		s.Nals = append(s.Nals, nals)
		s.Pkts = append(s.Pkts, s.curpkt)
	}
}

func (s *H264Record) Report(root string) {
	var fname string
	var w *os.File
	var err error
	var pid string = strconv.Itoa(s.Pid)
	var header string

	fname = filepath.Join(root, pid+".csv")
	w, err = os.Create(fname)
	if err != nil {
		panic(err)
	}
	header = "Pos, Size, PCR, PTS, DTS, (DTS-PCR)"
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
			strconv.FormatInt(p.Pts, 10),
			strconv.FormatInt(dts, 10),
			strconv.FormatInt(dts-pcr, 10),
		}
		fmt.Fprintln(w, strings.Join(cols, ", "))
	}
	w.Close()

	fname = filepath.Join(root, pid+"-nal"+".csv")
	w, err = os.Create(fname)
	if err != nil {
		panic(err)
	}
	header = "NAL units"
	fmt.Fprintln(w, header)
	for _, nals := range s.Nals {
		fmt.Fprintln(w, strings.Join(nals, ", "))
	}
	w.Close()
}
