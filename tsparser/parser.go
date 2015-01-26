package main

import (
	"llin/ts"
)

func parse(fname string, outdir string, psiOnly bool) {
	var pkts chan *ts.TsPkt

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
	records := make(map[int]ts.Record)
	for pid, s := range streams {
		records[pid] = ts.CreateRecord(pid, ts.GetStreamType(s))
	}

	pkts = ts.ParseFile(fname)
	for pkt := range pkts {
		if pcr, ok := pkt.PCR(); ok {
			if pids, ok := pcrs[pkt.Pid]; ok {
				for _, pid := range pids {
					records[pid].NotifyTime(pcr)
				}
			}
		}

		if record, ok := records[pkt.Pid]; ok {
			record.Process(pkt)
		}
	}

	for _, record := range records {
		record.Flush()
		record.Report(outdir)
	}
}
