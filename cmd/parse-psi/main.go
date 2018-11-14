package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"github.com/leonlinc/mpts/ts"
)

var (
	input string
)

type stat struct {
	previous_pcr_pos int64
	previous_pcr int64
	pat_pos_records []int64
}

func init() {
	flag.StringVar(&input, "i", "a.ts", "input file or udp stream")
}

func main() {
	flag.Parse()

	file, err := os.Open(input)
	if err != nil {
		log.Fatal(err)
	}
	parse(file)
}

func parse(file *os.File) {
	log.Println("Start parsing", file.Name())
	r := ts.NewReader(file)
	b := make([]byte, 188)
	s := stat{previous_pcr: -1, previous_pcr_pos: -1, pat_pos_records: nil}
	for {
		_, err := r.Read(b)
		if err != nil {
			if err != io.EOF {
				log.Fatal(err)
			}
			break
		}

		if r.PID == 0 {
			s.addPatPosRecord(r.Count())
		}
		if r.PCR() != -1 {
			pcr := r.PCR()
			pos := r.Count()
			fmt.Println("PCR", pos, pcr)
			if s.previous_pcr != -1 {
				for _, p := range s.pat_pos_records {
					t := s.previous_pcr + (p - s.previous_pcr_pos) * (pcr - s.previous_pcr) / (pos - s.previous_pcr_pos)
					fmt.Println("PAT", p, t)
				}
				s.pat_pos_records = nil
			}
			s.previous_pcr = pcr
			s.previous_pcr_pos = pos
		}
	}
	log.Println(s)
	log.Println("Finish parsing", file.Name())
}

func (s *stat) addPatPosRecord(pos int64) {
	s.pat_pos_records = append(s.pat_pos_records, pos)
}
