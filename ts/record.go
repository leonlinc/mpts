package ts

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Record interface {
	Process(pkt *TsPkt)
	NotifyTime(pcr int64)
	Flush()
	Report(root string)
}

type BaseRecord struct {
	PcrTime int64
}

func (b *BaseRecord) NotifyTime(pcr int64) {
	b.PcrTime = pcr
}

type PesRecord struct {
	BaseRecord
	Pid    int
	curpkt *PesPkt
	Pkts   []*PesPkt
}

func (s *PesRecord) Process(pkt *TsPkt) {
	if pkt.PUSI == 1 {
		if s.curpkt != nil {
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

	header := "Pos, Size, PCR, PTS, DTS, (DTS-PCR)"
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
}

type H264Record struct {
	BaseRecord
	Pid    int
	curpkt *PesPkt
	Pkts   []*PesPkt
	Nals   [][]string
}

func (s *H264Record) Process(pkt *TsPkt) {
	if pkt.PUSI == 1 {
		if s.curpkt != nil {
			nals := ParseNalUnits(s.curpkt.Data)
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

type Scte35Record struct {
	BaseRecord
	Pid      int
	CurByte  int64
	CurTime  int64
	CurData  []byte
	BytePos  []int64
	PcrTime  []int64
	Sections []*SpliceInfoSection
}

func (s *Scte35Record) Process(pkt *TsPkt) {
	if pkt.PUSI == 1 {
		if s.CurData != nil {
			section := ParseSpliceInfoSection(s.CurData)
			s.BytePos = append(s.BytePos, s.CurByte)
			s.PcrTime = append(s.PcrTime, s.CurTime)
			s.Sections = append(s.Sections, section)
		}
		s.CurByte = pkt.Pos
		s.CurTime = s.BaseRecord.PcrTime
		s.CurData = pkt.Data
	} else {
		s.CurData = append(s.CurData, pkt.Data...)
	}
}

func (s *Scte35Record) Flush() {
	if s.CurData != nil {
		section := ParseSpliceInfoSection(s.CurData)
		s.BytePos = append(s.BytePos, s.CurByte)
		s.PcrTime = append(s.PcrTime, s.CurTime)
		s.Sections = append(s.Sections, section)
	}
}

func (s *Scte35Record) Report(root string) {
	fname := filepath.Join(root, strconv.Itoa(s.Pid)+".csv")
	w, err := os.Create(fname)
	if err != nil {
		panic(err)
	}
	defer w.Close()

	fmt.Fprintln(w, "pos, pcr, type, pts_time, pts_adjust, duration")
	if s.Sections != nil {
		for i, section := range s.Sections {
			splice_type := section.GetSpliceType()
			pts, adj := section.GetSpliceTime()
			duration := section.GetSpliceDuration()
			fmt.Fprintf(w, "%v, %v, %v, %v, %v, %v\n",
				s.BytePos[i],
				s.PcrTime[i]/300,
				splice_type,
				pts,
				adj,
				duration)
		}
	}
}

func CreateRecord(pid int, t string) Record {
	var record Record
	switch t {
	case "SCTE-35":
		record = &Scte35Record{Pid: pid}
	case "MPEG-4 AVC Video":
		record = &H264Record{Pid: pid}
	default:
		record = &PesRecord{Pid: pid}
	}
	return record
}
