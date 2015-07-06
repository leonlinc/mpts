package ts

import (
	"fmt"
	"time"
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
		pkt.AdaptField = &AdaptField{}
		aflen := r.ReadBit(8)
		if aflen > 0 {
			flags := r.ReadBit(8)
			if (flags & 0x10) != 0 {
				pkt.hasPCR = true
				pkt.pcr = ParsePcr(r)
			}
			if (flags & 0x08) != 0 {
				ParsePcr(r) // OPCR
			}
			if (flags & 0x04) != 0 {
				r.ReadBit(8)
			}
			if (flags & 0x02) != 0 {
				privLen := r.ReadBit(8)
				pkt.AdaptField.PrivateData = r.Data[r.Base : r.Base+privLen]
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

type AdaptField struct {
	PrivateData []byte
}

type AuInfo struct {
	CodingFormat         int
	CodingType           int
	RefPicIdc            int
	PicStruct            int
	PtsPresent           bool
	ProfileInfoPresent   bool
	StreamInfoPresent    bool
	TrickModeInfoPresent bool
	Pts                  int64
	// stream info
	AuFrameRateCode int
	// profile info
	AuProfile  int
	AuAvcFlags int
	AuLevel    int
}

type DirecTvTimeCode struct {
	DropFrameFlag bool
	Hours         int
	Minutes       int
	Seconds       int
	Pictures      int
}

type BroadcastId struct {
	Identifier         int
	Origin             int
	ServiceName        string
	TransportStreamId  int
	MajorChannelNumber int
	MinorChannelNumber int
}

type EBP struct {
	Standard     string
	Fragment     bool // ENC_bound_pt
	Segment      bool
	UtcTime      *string
	UtcTimestamp *int64
}

type AdaptFieldPrivData struct {
	FieldTag byte
	FieldLen byte
	*AuInfo
	*DirecTvTimeCode
	*BroadcastId
	*EBP
}

func NTPTimeToUnixTime(ntpTime int64) time.Time {
	var sec, usec int64
	sec = int64((uint64(ntpTime) >> 32) - 0x83AA7E80) // the seconds from Jan 1, 1900 to Jan 1, 1970
	usec = int64(float64(uint32(ntpTime)) * 1.0e6 / float64(int64(1)<<32))
	return time.Unix(sec, usec*1000)
}

func ParseAdaptFieldPrivData(data []byte) []AdaptFieldPrivData {
	var privList []AdaptFieldPrivData
	for len(data) > 0 {
		priv := AdaptFieldPrivData{}
		priv.FieldTag = data[0]
		priv.FieldLen = data[1]
		if priv.FieldTag == 0x02 {
			r := &Reader{Data: data[2:]}

			auInfo := &AuInfo{}
			auInfo.CodingFormat = r.ReadBit(4)
			auInfo.CodingType = r.ReadBit(4)

			if priv.FieldLen > 1 {
				auInfo.RefPicIdc = r.ReadBit(2)
				auInfo.PicStruct = r.ReadBit(2)
				auInfo.PtsPresent = r.ReadBit(1) != 0
				auInfo.ProfileInfoPresent = r.ReadBit(1) != 0
				auInfo.StreamInfoPresent = r.ReadBit(1) != 0
				auInfo.TrickModeInfoPresent = r.ReadBit(1) != 0
			}

			if auInfo.PtsPresent {
				auInfo.Pts = r.ReadBit64(32)
			}

			if auInfo.StreamInfoPresent {
				r.SkipBit(4)
				auInfo.AuFrameRateCode = r.ReadBit(4)
			}

			if auInfo.ProfileInfoPresent {
				auInfo.AuProfile = r.ReadBit(8)
				auInfo.AuAvcFlags = r.ReadBit(8)
				auInfo.AuLevel = r.ReadBit(8)
			}

			priv.AuInfo = auInfo
		} else if priv.FieldTag == 0xA0 {
			r := &Reader{Data: data[2:]}

			tcInfo := &DirecTvTimeCode{}
			tcInfo.DropFrameFlag = r.ReadBit(1) != 0
			tcInfo.Hours = r.ReadBit(5)
			tcInfo.Minutes = r.ReadBit(6)
			tcInfo.Seconds = r.ReadBit(6)
			tcInfo.Pictures = r.ReadBit(6)
			priv.DirecTvTimeCode = tcInfo
		} else if priv.FieldTag == 0xAD {
			r := &Reader{Data: data[2:]}
			biInfo := &BroadcastId{}
			biInfo.Identifier = r.ReadBit(32)
			biInfo.Origin = r.ReadBit(8)
			biInfo.ServiceName = string(r.Data[r.Base : r.Base+14])
			r.SkipByte(14)
			biInfo.TransportStreamId = r.ReadBit(16)
			if biInfo.Origin == 1 {
				r.ReadBit(4)
				biInfo.MajorChannelNumber = r.ReadBit(10)
				biInfo.MinorChannelNumber = r.ReadBit(10)
			}
			priv.BroadcastId = biInfo
		} else if priv.FieldTag == 0xA9 {
			// pre-standard EBP
			r := &Reader{Data: data[2:]}
			if priv.FieldLen > 1 {
				ebp := &EBP{}
				ebp.Standard = "PreStandard" // Comcast
				ebp.Fragment = r.ReadBit(1) != 0
				ebp.Segment = r.ReadBit(1) != 0
				var sap = r.ReadBit(1)
				var grouping = r.ReadBit(1)
				var time = r.ReadBit(1)
				r.ReadBit(1)
				r.ReadBit(1)
				var ext = r.ReadBit(1)
				if ext == 1 {
					r.ReadBit(8)
				}
				if sap == 1 {
					r.ReadBit(8)
				}
				if grouping == 1 {
					r.ReadBit(8)
				}
				if time == 1 {
					ebp.UtcTimestamp = new(int64)
					*ebp.UtcTimestamp = r.ReadBit64(64)
					ebp.UtcTime = new(string)
					*ebp.UtcTime = NTPTimeToUnixTime(*ebp.UtcTimestamp).String()
				}
				priv.EBP = ebp
			}
		} else if priv.FieldTag == 0xDF {
			// cablelab EBP
			r := &Reader{Data: data[2:]}
			if priv.FieldLen > 1 {
				ebp := &EBP{}
				ebp.Standard = "CableLab" // OpenCable
				r.ReadBit(32)
				ebp.Fragment = r.ReadBit(1) != 0
				ebp.Segment = r.ReadBit(1) != 0
				var sap = r.ReadBit(1)
				var grouping = r.ReadBit(1)
				var time = r.ReadBit(1)
				r.ReadBit(1)
				r.ReadBit(1)
				var ext = r.ReadBit(1)
				var extPartition = 0
				if ext == 1 {
					extPartition = r.ReadBit(1)
					r.ReadBit(7)
				}
				if sap == 1 {
					r.ReadBit(8)
				}
				if grouping == 1 {
					var extFlag = 1
					for extFlag == 1 {
						extFlag = r.ReadBit(1)
						r.ReadBit(7)
					}
				}
				if time == 1 {
					ebp.UtcTimestamp = new(int64)
					*ebp.UtcTimestamp = r.ReadBit64(64)
					ebp.UtcTime = new(string)
					*ebp.UtcTime = NTPTimeToUnixTime(*ebp.UtcTimestamp).String()
				}
				if extPartition == 1 {
					r.ReadBit(8)
				}
				priv.EBP = ebp
			}
		}
		privList = append(privList, priv)
		data = data[2+priv.FieldLen:]
	}
	return privList
}

type TsPkt struct {
	SyncByte int
	PUSI     int
	Pid      int
	CC       int
	*AdaptField
	Data   []byte
	pcr    int64
	hasPCR bool
	Pos    int64
}

func (p TsPkt) PCR() (int64, bool) {
	if p.hasPCR == true {
		return p.pcr, true
	}
	return 0, false
}

type PesPkt struct {
	Pos      int64
	Size     int64
	StreamId int
	Pcr      int64
	PcrPos   int64
	Pts      int64
	Dts      int64
	Data     []byte
}

func (p *PesPkt) Read(pkt *TsPkt) (n int) {
	r := &Reader{Data: pkt.Data}

	r.SkipByte(3)
	streamId := r.ReadBit(8)
	p.StreamId = streamId
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
	case streamId == 0xBE:
		// Padding stream
	default:
		fmt.Println("Unknown stream id", streamId)
	}

	return
}
