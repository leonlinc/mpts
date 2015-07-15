package ts

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type AdaptOut struct {
	Pos     int64
	Content AdaptFieldPrivData
}

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
	AdaptFieldPrivDataLog *os.File
}

func (b *BaseRecord) NotifyTime(pcr int64, pos int64) {
	b.PcrTime = pcr
	b.PcrPos = pos
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

func (logger *BaseRecord) LogAdaptFieldPrivData(pkt *TsPkt) {
	if pkt.AdaptField == nil || pkt.AdaptField.PrivateData == nil {
		return
	}
	data := pkt.AdaptField.PrivateData
	if logger.AdaptFieldPrivDataLog == nil {
		fname := filepath.Join(logger.Root, strconv.Itoa(logger.Pid)+"-tspriv.csv")
		var err error
		logger.AdaptFieldPrivDataLog, err = os.Create(fname)
		if err != nil {
			panic(err)
		}
	}
	privList := ParseAdaptFieldPrivData(data)
	for _, p := range privList {
		adaptOut := AdaptOut{pkt.Pos, p}
		c, _ := json.Marshal(adaptOut)
		fmt.Fprintln(logger.AdaptFieldPrivDataLog, string(c))
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

