package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

const (
	MtuSize = 1500
	UdpSize = 1316
)

var dumpFlag = flag.Bool("dump", false, "dump TS file")
var loopFlag = flag.Bool("loop", true, "loop the source")
var outFile = flag.String("out", "out.ts", "output file name")
var sendFlag = flag.Bool("send", false, "send TS file")

func main() {
	flag.Parse()
	args := flag.Args()

	if *dumpFlag {
		if len(args) != 3 {
			fmt.Printf("Usage: tsparser -dump interface address port\n")
			os.Exit(1)
		}

		dump(args[0], args[1], args[2], *outFile)
	} else if *sendFlag {
		if len(args) != 5 {
			fmt.Printf("Usage: tsparser -send ip address port file bitrate\n")
			os.Exit(1)
		}
		send(args[0], args[1], args[2], args[3], args[4])
	} else {
		if len(args) != 1 {
			fmt.Printf("Usage: tsparser [options] arguments\n")
			os.Exit(1)
		}
		parseFile(args[0])
	}
}

func dump(ifiName, addr, port, output string) {
	ifi, err := net.InterfaceByName(ifiName)
	if err != nil {
		log.Fatal(err)
	}

	udpAddr, err := net.ResolveUDPAddr("udp", addr+":"+port)
	if err != nil {
		log.Fatal(err)
	}

	conn, err := net.ListenMulticastUDP("udp", ifi, udpAddr)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	f, err := os.Create(output)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	b := make([]byte, MtuSize)
	for {
		n, err := conn.Read(b)
		if err != nil {
			log.Fatal(err)
		}
		offset := n - UdpSize
		f.Write(b[offset:n])
	}
}

func send(ip, addr, port, input, br string) {
	f, err := os.Open(input)
	if err != nil {
		log.Fatalln(err)
	}
	defer f.Close()

	bitrate, err := strconv.ParseInt(br, 10, 64)
	if err != nil {
		log.Fatalln(err)
	}

	serverAddr, err := net.ResolveUDPAddr("udp", addr+":"+port)
	if err != nil {
		log.Fatalln(err)
	}

	clientAddr, err := net.ResolveUDPAddr("udp", ip+":")
	if err != nil {
		log.Fatalln(err)
	}

	conn, err := net.DialUDP("udp", clientAddr, serverAddr)
	if err != nil {
		log.Fatalln(err)
	}
	defer conn.Close()

	log.Println("start sending at", conn.LocalAddr())

	var accBits int64
	buf := make([]byte, UdpSize)
	startTime := time.Now()
	timeout := time.Tick(time.Millisecond)

Loop:
	for _ = range timeout {
		for {
			elapsedTime := int64(time.Since(startTime) / time.Millisecond)
			if accBits >= bitrate*elapsedTime/1000 {
				break
			}

			n, err := f.Read(buf)
			if n != 0 {
				accBits += int64(n * 8)
				_, err = conn.Write(buf[:n])
				if err != nil {
					log.Fatalln(err)
				}
			}

			// Handle loop point
			if err != nil {
				if err == io.EOF {
					if *loopFlag {
						log.Println("source loop point")
						f.Seek(0, 0)
						continue
					} else {
						break Loop
					}
				} else {
					log.Fatalln(err)
				}
			}
		}
	}
}

func parseFile(input string) {
	outdir := filepath.Base(input) + ".log"
	os.Mkdir(outdir, os.ModeDir|0755)
	parse(input, outdir, false)
}
