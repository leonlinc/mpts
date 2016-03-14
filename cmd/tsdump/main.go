package main

import (
	"fmt"
	"log"
	"net"
	"os"
)

const (
	MaxSize  = 1500
	UDPSize  = 1316
	HRTPSize = 1360
)

func main() {
	if len(os.Args) != 4 {
		fmt.Printf("Usage: %s interface addr port\n", os.Args[0])
		os.Exit(0)
	}

	ifiName, addr := os.Args[1], os.Args[2]+":"+os.Args[3]
	fmt.Printf("Dumping %s@%s\n", addr, ifiName)

	ifi, err := net.InterfaceByName(ifiName)
	if err != nil {
		log.Fatal(err)
	}

	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		log.Fatal(err)
	}

	conn, err := net.ListenMulticastUDP("udp", ifi, udpAddr)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	f, err := os.Create("out.ts")
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
		if n == UDPSize {
			f.Write(b[:UDPSize])
		} else if n == HRTPSize {
			offset := HRTPSize - UDPSize
			f.Write(b[offset:HRTPSize])
		}
	}
}
