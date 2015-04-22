package main

import (
	"github.com/xjbt/ts"
)

func parse(fname string, outdir string, psiOnly bool) {
	var pkts chan *ts.TsPkt

	// PCR PID -> PCR values
	var progPcrList = make(map[int][]ts.PcrInfo)

	pkts = ts.ParseFile(fname)
	psiParser := ts.NewPsiParser()
	for pkt := range pkts {
		if ok := psiParser.Parse(pkt); ok {
			psiParser.Report(outdir)
			break
		}
	}

	if psiOnly {
		return
	}

	streams := psiParser.GetStreams()

	pcrs := psiParser.GetPcrs()
	for pcrPid, _ := range pcrs {
		// Default PCR list length: 1500 = 25Hz * 60s
		progPcrList[pcrPid] = make([]ts.PcrInfo, 0)
	}

	records := make(map[int]ts.Record)
	for pid, s := range streams {
		records[pid] = ts.CreateRecord(pid, ts.GetStreamType(s))
	}

	pkts = ts.ParseFile(fname)
	for pkt := range pkts {
		if pcr, ok := pkt.PCR(); ok {
			if pids, ok := pcrs[pkt.Pid]; ok {
				// Save the PCR value
				progPcrList[pkt.Pid] = append(
					progPcrList[pkt.Pid],
					ts.PcrInfo{pkt.Pos, pcr})
				for _, pid := range pids {
					records[pid].NotifyTime(pcr, pkt.Pos)
				}
			}
		}

		if record, ok := records[pkt.Pid]; ok {
			record.Process(pkt)
		}
	}

	for pcrPid, pcrList := range progPcrList {
		ts.CheckPcrInterval(outdir, pcrPid, pcrList)
	}

	for _, record := range records {
		record.Flush()
		record.Report(outdir)
	}
}
