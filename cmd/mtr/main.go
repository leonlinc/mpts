package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
)

const (
	MaxSize = 1500
	UDPSize = 1316
)

var dumpFlag = flag.Bool("dump", false, "")
var outFile = flag.String("out", "out.ts", "")

func main() {
	flag.Parse()

	if *dumpFlag {
		args := flag.Args()
		if len(args) != 3 {
			fmt.Printf("Usage: mtr -dump interface address port\n")
			os.Exit(1)
		}

		dump(args[0], args[1], args[2], *outFile)
	} else {

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

	b := make([]byte, MaxSize)
	for {
		n, err := conn.Read(b)
		if err != nil {
			log.Fatal(err)
		}
		offset := n - UDPSize
		f.Write(b[offset:n])
	}
}
