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
	NotifyTime(pcr int64, pos int64)
	Flush()
	Report(root string)
}

type BaseRecord struct {
	Root      string
	Pid       int
	PcrTime   int64
	PcrPos    int64
	IFrameLog *os.File
}

func (b *BaseRecord) NotifyTime(pcr int64, pos int64) {
	b.PcrTime = pcr
	b.PcrPos = pos
}

type PesRecord struct {
	BaseRecord
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
		s.curpkt.PcrPos = s.BaseRecord.PcrPos
		var startcode = []byte{0, 0, 1}
		if 0 == bytes.Compare(startcode, pkt.Data[0:3]) {
			hlen := s.curpkt.Read(pkt)
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

type Scte35Record struct {
	BaseRecord
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

func CreateRecord(pid int, t string, root string) Record {
	var record Record
	switch t {
	case "SCTE-35":
		record = &Scte35Record{BaseRecord: BaseRecord{Pid: pid}}
	case "MPEG-4 AVC Video":
		record = &H264Record{BaseRecord: BaseRecord{Pid: pid, Root: root}}
	case "MPEG-2 Video":
		record = &Mp2vRecord{BaseRecord: BaseRecord{Pid: pid, Root: root}}
	default:
		record = &PesRecord{BaseRecord: BaseRecord{Pid: pid}}
	}
	return record
}

func (r *BaseRecord) LogIFrame(i IFrameInfo) {
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
