package ts

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Mp2vUserData struct {
	Pos     int64
	AFD     *int
	Caption bool
	Bar     bool
}

type Mp2vTimeCode struct {
	DropFrameFlag int
	Hours         int
	Minutes       int
	Seconds       int
	Pictures      int
}

type Mp2vHeaders struct {
	*Mp2vGopHeader
	*Mp2vPicHeader
	UserData []*Mp2vUserData
}

type Mp2vGopHeader struct {
	Mp2vTimeCode
	ClosedGop int
}

type Mp2vPicHeader struct {
	TemporalReference int
	PictureCodingType int
}

func ParseATSC(data []byte) (cc bool, bar bool) {
	code := data[0]
	if code == 0x03 {
		cc = true
	} else if code == 0x06 {
		bar = true
	}
	return
}

func ParseAFD(data []byte) (int, error) {
	if data[0] == 0x41 {
		format := data[1] & 0x0F
		return int(format), nil
	}
	return 0, errors.New("Active format does not exist")
}

func ParseMp2vUserData(data []byte) *Mp2vUserData {
	var result = &Mp2vUserData{}
	var idATSC = []byte("GA94") // 0x47413934
	var idAFD = []byte("DTG1")  // 0x44544731
	if bytes.Compare(idATSC, data[0:4]) == 0 {
		result.Caption, result.Bar = ParseATSC(data[4:])
	} else if bytes.Compare(idAFD, data[0:4]) == 0 {
		afd, err := ParseAFD(data[4:])
		if err == nil {
			result.AFD = &afd
		}
	}
	return result
}

func ParseMp2vHeaders(data []byte) Mp2vHeaders {
	var result Mp2vHeaders
	var pos int
	var startcode = []byte{0, 0, 1}
	var startlen = len(startcode)
	for pos+startlen+1 < len(data) {
		if bytes.Compare(startcode, data[pos:pos+startlen]) == 0 {
			pos += startlen
			code := int(data[pos])
			elem := data[pos+1:]
			if code == 0xB2 {
				userData := ParseMp2vUserData(elem)
				result.UserData = append(result.UserData, userData)
			} else if code == 0x00 {
				result.Mp2vPicHeader = ParseMp2vPicHeader(elem)
			} else if code == 0xB8 {
				result.Mp2vGopHeader = ParseMp2vGopHeader(elem)
			}
		}
		pos += 1
	}
	return result
}

func ParseMp2vPicHeader(data []byte) *Mp2vPicHeader {
	r := &Reader{Data: data}
	h := &Mp2vPicHeader{}
	h.TemporalReference = r.ReadBit(10)
	h.PictureCodingType = r.ReadBit(3)
	return h
}

func ParseMp2vGopHeader(data []byte) *Mp2vGopHeader {
	r := &Reader{Data: data}
	h := &Mp2vGopHeader{}
	h.Mp2vTimeCode.DropFrameFlag = r.ReadBit(1)
	h.Mp2vTimeCode.Hours = r.ReadBit(5)
	h.Mp2vTimeCode.Minutes = r.ReadBit(6)
	r.ReadBit(1)
	h.Mp2vTimeCode.Seconds = r.ReadBit(6)
	h.Mp2vTimeCode.Pictures = r.ReadBit(6)
	h.ClosedGop = r.ReadBit(1)
	return h
}

type Mp2vRecord struct {
	BaseRecord
	Root      string
	Pid       int
	curpkt    *PesPkt
	Pkts      []*PesPkt
	UserData  []*Mp2vUserData
	IFrameLog *os.File
}

type IFrameInfo struct {
	Pos int64
	Pts int64
	Key bool
}

func (r *Mp2vRecord) LogIFrame(i IFrameInfo) {
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

func (s *Mp2vRecord) Process(pkt *TsPkt) {
	if pkt.PUSI == 1 {
		if s.curpkt != nil {
			headers := ParseMp2vHeaders(s.curpkt.Data)
			if headers.Mp2vPicHeader.PictureCodingType == 1 {
				i := IFrameInfo{}
				i.Pos = s.curpkt.Pos
				i.Pts = s.curpkt.Pts
				if headers.Mp2vGopHeader != nil {
					i.Key = headers.Mp2vGopHeader.ClosedGop == 1
				}
				s.LogIFrame(i)
			}
			userData := headers.UserData
			for _, u := range userData {
				u.Pos = s.curpkt.Pos
			}
			s.UserData = append(s.UserData, userData...)
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

func (s *Mp2vRecord) Flush() {
	if s.curpkt != nil {
		s.Pkts = append(s.Pkts, s.curpkt)
	}
}

func (s *Mp2vRecord) Report(root string) {
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

	fname = filepath.Join(root, pid+"-userdata"+".csv")
	w, err = os.Create(fname)
	if err != nil {
		panic(err)
	}
	header = "User Data"
	fmt.Fprintln(w, header)
	for _, userData := range s.UserData {
		c, _ := json.Marshal(userData)
		fmt.Fprintln(w, string(c))
	}
	w.Close()
}
