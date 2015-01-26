package main

import (
	//"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
)

var index int

func init() {
	log.SetFlags(log.Lshortfile)

	flag.IntVar(&index, "index", 0, "interface index")
	flag.Usage = func() {
		flag.PrintDefaults()
	}
}

func main() {
	flag.Parse()
	if flag.NArg() < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] ip port\n", os.Args[0])
		flag.Usage()
		return
	}

	ip := flag.Arg(0)
	ps, pe := parsePort(flag.Arg(1))

	cs := make([]chan bool, pe-ps+1)
	i := 0
	for port := ps; port <= pe; port++ {
		c := receive(ip, strconv.FormatInt(port, 10))
		cs[i] = c
		i += 1
	}

	// Wait for all receiver to stop
	for _, c := range cs {
		<-c
	}
}

func parsePort(portRange string) (int64, int64) {
	var start, end int64
	var err error
	ports := strings.Split(portRange, "-")
	start, err = strconv.ParseInt(ports[0], 10, 16)
	if err != nil {
		log.Fatalln(err)
	}
	if len(ports) > 1 {
		end, err = strconv.ParseInt(ports[1], 10, 16)
		if err != nil {
			log.Fatalln(err)
		}
	} else {
		end = start
	}
	return start, end
}

func receive(ip, port string) chan bool {
	c := make(chan bool)
	go func() {
		addr := ip + ":" + port
		gaddr, err := net.ResolveUDPAddr("udp", addr)
		if err != nil {
			log.Fatalln(err)
		}

		var ifi *net.Interface
		if index != 0 {
			ifi, err = net.InterfaceByIndex(index)
			if err != nil {
				log.Fatalln(err)
			}
		}

		conn, err := net.ListenMulticastUDP("udp", ifi, gaddr)
		if err != nil {
			log.Fatalln(err)
		}
		defer conn.Close()
		fmt.Println("Local address:", conn.LocalAddr())

		file, err := os.Create("output-" + port + ".ts")
		if err != nil {
			log.Fatalln(err)
		}
		defer file.Close()

		buf := make([]byte, 1500)
		for {
			n, err := conn.Read(buf)
			if err != nil {
				log.Fatalln(err)
				break
			}
			pre := n % 188
			file.Write(buf[pre:n])
		}
	}()
	return c
}
