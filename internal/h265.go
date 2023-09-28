package mpts

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

var HevcNalUnitType []string = []string{
	"trail_n",
	"trail_r",
	"tsa_n",
	"tsa_r",
	"stsa_n",
	"stsa_r",
	"radl_n",
	"radl_r",
	"rasl_n",
	"rasl_r",
	"rsv_vcl_n10",
	"rsv_vcl_r11",
	"rsv_vcl_n12",
	"rsv_vcl_r13",
	"rsv_vcl_n14",
	"rsv_vcl_r15",
	"bla_w_lp",
	"bla_w_radl",
	"bla_n_lp",
	"idr_w_radl",
	"idr_n_lp",
	"cra_nut",
	"rsv_irap_vcl22",
	"rsv_irap_vcl23",
	"rsv_vcl24",
	"rsv_vcl25",
	"rsv_vcl26",
	"rsv_vcl27",
	"rsv_vcl28",
	"rsv_vcl29",
	"rsv_vcl30",
	"rsv_vcl31",
	"vps_nut",
	"sps_nut",
	"pps_nut",
	"aud_nut",
	"eos_nut",
	"eob_nut",
	"fd_nut",
	"prefix_sei_nut",
	"suffix_sei_nut",
	"rsv_nvcl41",
	"rsv_nvcl42",
	"rsv_nvcl43",
	"rsv_nvcl44",
	"rsv_nvcl45",
	"rsv_nvcl46",
	"rsv_nvcl47",
	"unspec48",
	"unspec49",
	"unspec50",
	"unspec51",
	"unspec52",
	"unspec53",
	"unspec54",
	"unspec55",
	"unspec56",
	"unspec57",
	"unspec58",
	"unspec59",
	"unspec60",
	"unspec61",
	"unspec62",
	"unspec63",
}

func GetHevcNalUnitType(b int) string {
	b = (b & 0x7E) >> 1
	if b <= len(HevcNalUnitType) {
		return HevcNalUnitType[b]
	} else {
		return HevcNalUnitType[0]
	}
}

func ParseHevcNalUnits(data []byte) []string {
	var nals []string
	var pos int
	var startcode = []byte{0, 0, 1}
	var startlen = len(startcode)
	for pos+5 < len(data) {
		if bytes.Compare(startcode, data[pos:pos+startlen]) == 0 {
			pos += startlen
			nal := GetHevcNalUnitType(int(data[pos]))
			nals = append(nals, nal)
		}
		pos += 1
	}
	return nals
}

type H265Record struct {
	BaseRecord
	curpkt *PesPkt
	Pkts   []*PesPkt
	Nals   [][]string
	// Workaround PES parsing error
	WorkaroundPESFlag bool
	WorkaroundPES     []byte
}

const minHevcPesHeaderLen = 19

func (s *H265Record) Process(pkt *TsPkt) {
	s.LogAdaptFieldPrivData(pkt)
	if pkt.PUSI == 1 {
		if s.curpkt != nil {
			nals := ParseHevcNalUnits(s.curpkt.Data)
			for _, nal := range nals {
				if nal == "idr_w_radl" || nal == "idr_n_lp" {
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

		if len(pkt.Data) >= minHevcPesHeaderLen {
			var startcode = []byte{0, 0, 1}
			if 0 == bytes.Compare(startcode, pkt.Data[0:3]) {
				hlen := s.curpkt.Read(pkt.Data)
				pkt.Data = pkt.Data[hlen:]
			} else {
				log.Println("PES start code error")
			}
		} else {
			log.Println("Workaround for pkt:", pkt.Pos, "size:", len(pkt.Data))
			s.WorkaroundPESFlag = true
			s.WorkaroundPES = nil
		}
	}

	if s.WorkaroundPESFlag {
		s.WorkaroundPES = append(s.WorkaroundPES, pkt.Data...)
		pkt.Data = nil
		if len(s.WorkaroundPES) >= minHevcPesHeaderLen {
			var startcode = []byte{0, 0, 1}
			if 0 == bytes.Compare(startcode, s.WorkaroundPES[0:3]) {
				hlen := s.curpkt.Read(s.WorkaroundPES)
				pkt.Data = s.WorkaroundPES[hlen:]
				s.WorkaroundPESFlag = false
			} else {
				log.Println("PES start code error")
			}
		}
	}

	if s.curpkt != nil {
		s.curpkt.Size += int64(len(pkt.Data))
		s.curpkt.Data = append(s.curpkt.Data, pkt.Data...)
	}
}

func (s *H265Record) Flush() {
	if s.curpkt != nil {
		nals := ParseHevcNalUnits(s.curpkt.Data)
		for _, nal := range nals {
			if nal == "idr_w_radl" || nal == "idr_n_lp" {
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
}

func (s *H265Record) Report(root string) {
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
