package ts

import (
	"fmt"
)

func ParseTsPkt(data []byte) *TsPkt {
	pkt := &TsPkt{}
	r := &Reader{Data: data}

	pkt.SyncByte = r.ReadBit(8)
	r.SkipBit(1)
	pkt.PUSI = r.ReadBit(1)
	r.SkipBit(1)
	pkt.Pid = r.ReadBit(13)
	r.SkipBit(2)
	afctrl := r.ReadBit(2)
	pkt.CC = r.ReadBit(4)
	if afctrl == 2 || afctrl == 3 {
		aflen := r.ReadBit(8)
		if aflen > 0 {
			flags := r.ReadBit(8)
			if (flags & 0x10) != 0 {
				pkt.hasPCR = true
				pkt.pcr = ParsePcr(r)
			}
		}
		pkt.Data = data[4+1+aflen:]
	} else {
		pkt.Data = data[4:]
	}

	return pkt
}

func ParsePts(r *Reader) int64 {
	var pts int64
	r.ReadBit(4)
	pts = r.ReadBit64(3)
	pts <<= 15
	r.ReadBit(1)
	pts += r.ReadBit64(15)
	pts <<= 15
	r.ReadBit(1)
	pts += r.ReadBit64(15)
	r.ReadBit(1)
	return pts
}

func ParsePcr(r *Reader) int64 {
	base := r.ReadBit64(33)
	r.SkipBit(6)
	ext := r.ReadBit64(9)
	return base*300 + ext
}

type TsPkt struct {
	SyncByte int
	PUSI     int
	Pid      int
	CC       int
	Data     []byte
	pcr      int64
	hasPCR   bool
	Pos      int64
}

func (p TsPkt) PCR() (int64, bool) {
	if p.hasPCR == true {
		return p.pcr, true
	}
	return 0, false
}

type PesPkt struct {
	Pos  int64
	Size int64
	Pcr  int64
	Pts  int64
	Dts  int64
	Data []byte
}

func (p *PesPkt) Read(pkt *TsPkt) (n int) {
	r := &Reader{Data: pkt.Data}

	r.SkipByte(3)
	streamId := r.ReadBit(8)
	r.SkipByte(2)
	switch {
	case streamId >= 0xC0 && streamId < 0xF0:
		fallthrough
	case streamId == 0xBD:
		r.SkipByte(1)
		flags := r.ReadBit(2)
		r.SkipBit(6)
		r.SkipByte(1)
		n = 9
		if flags == 2 {
			p.Pts = ParsePts(r)
			n += 5
		} else if flags == 3 {
			p.Pts = ParsePts(r)
			p.Dts = ParsePts(r)
			n += 10
		}
	default:
		fmt.Println("Unknown stream id", streamId)
	}

	return
}
