package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/leonlinc/mpts/internal/ts"
)

var (
	input string
)

type stat struct {
	init            bool
	previous_pcr    ts.PcrRecord
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
	s := stat{init: false}
	for {
		_, err := r.Read(b)
		if err != nil {
			if err != io.EOF {
				log.Fatal(err)
			}
			break
		}

		if r.PID == 0 {
			s.addPatPosRecord(r.Pos())
		}
		if pcr, ok := r.Pcr(); ok {
			fmt.Println("PCR", pcr)
			if s.init == false {
				for _, p := range s.pat_pos_records {
					t := s.previous_pcr.Pcr + (p-s.previous_pcr.Pos)*(pcr.Pcr-s.previous_pcr.Pcr)/(pcr.Pos-s.previous_pcr.Pos)
					fmt.Println("PAT", p, t)
				}
				s.pat_pos_records = nil
			}
			s.previous_pcr = pcr
		}
	}
	log.Println(s)
	log.Println("Finish parsing", file.Name())
}

func (s *stat) addPatPosRecord(pos int64) {
	s.pat_pos_records = append(s.pat_pos_records, pos)
}
