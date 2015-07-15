package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"time"
)

var loop bool
var ifi string

func init() {
	log.SetFlags(log.Lshortfile)

	flag.BoolVar(&loop, "loop", false, "loop play the file")
	flag.StringVar(&ifi, "i", "", "interface")
	flag.Usage = func() {
		flag.PrintDefaults()
	}
}

func main() {
	flag.Parse()
	if flag.NArg() < 4 {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] file bitrate ip port\n", os.Args[0])
		flag.Usage()
		return
	}

	file, err := os.Open(flag.Arg(0))
	if err != nil {
		log.Fatalln(err)
	}
	defer file.Close()

	bitrate, err := strconv.ParseFloat(flag.Arg(1), 64)
	if err != nil {
		log.Fatalln(err)
	}

	maddr := flag.Arg(2) + ":" + flag.Arg(3)
	raddr, err := net.ResolveUDPAddr("udp", maddr)
	if err != nil {
		log.Fatalln(err)
	}

	var laddr *net.UDPAddr
	if ifi != "" {
		laddr, err = net.ResolveUDPAddr("udp", ifi+":")
		if err != nil {
			log.Fatalln(err)
		}
	}

	conn, err := net.DialUDP("udp", laddr, raddr)
	if err != nil {
		log.Fatalln(err)
	}
	defer conn.Close()
	fmt.Println("Local address:", conn.LocalAddr())

	s := newSender(file, conn, bitrate, loop)
	c := time.Tick(1 * time.Millisecond)
	for _ = range c {
		err = s.send()
		if err != nil {
			if err == io.EOF {
				break
			} else {
				log.Fatalln(err)
			}
		}
	}
}

type sender struct {
	file    *os.File
	conn    *net.UDPConn
	bitrate float64
	loop    bool
	start   time.Time
	acc     int64
	buf     []byte
}

func newSender(file *os.File, conn *net.UDPConn, bitrate float64, loop bool) *sender {
	size := 188 * 7
	b := make([]byte, size)
	return &sender{file, conn, bitrate, loop, time.Now(), 0, b}
}

func (s *sender) send() error {
	for {
		// Control the sending bitrate
		if float64(s.acc*8) >= s.bitrate*time.Since(s.start).Seconds() {
			break
		}

		n, err := s.file.Read(s.buf)
		if err != nil {
			if err == io.EOF {
				// Loop play the file
				if s.loop == true {
					s.file.Seek(0, 0)
					continue
				}
			}
			return err
		}

		s.acc += int64(n)
		_, err = s.conn.Write(s.buf[:n])
		if err != nil {
			return err
		}
	}
	return nil
}
