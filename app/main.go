package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"strings"
)

var _ = net.ListenUDP

type header struct {
	id      uint16
	flags   uint16
	qdcount uint16
	ancount uint16
	nscount uint16
	arcount uint16
}

type question struct {
	name  []byte
	qtype uint16
	class uint16
}

func makeLabel(domain string) []byte {
	var label []byte
	for _, part := range strings.Split(domain, ".") {
		label = append(label, byte(len(part)))
		label = append(label, []byte(part)...)
	}
	label = append(label, 0)
	return label
}

func main() {
	udpAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:2053")
	if err != nil {
		fmt.Println("Failed to resolve UDP address:", err)
		return
	}

	udpConn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		fmt.Println("Failed to bind to address:", err)
		return
	}
	defer udpConn.Close()

	buf := make([]byte, 512)

	for {
		size, source, err := udpConn.ReadFromUDP(buf)
		if err != nil {
			fmt.Println("Error receiving data:", err)
			break
		}

		receivedData := string(buf[:size])
		fmt.Printf("Received %d bytes from %s: %s\n", size, source, receivedData)

		h := header{
			id:      1234,
			flags:   0x8000,
			qdcount: 1,
			ancount: 0,
			nscount: 0,
			arcount: 0,
		}
		q := question{
			name:  makeLabel("codecrafters.io"),
			qtype: 1,
			class: 1,
		}
		buf := new(bytes.Buffer)
		binary.Write(buf, binary.BigEndian, h)
		binary.Write(buf, binary.BigEndian, q.name)
		binary.Write(buf, binary.BigEndian, q.qtype)
		binary.Write(buf, binary.BigEndian, q.class)
		response := buf.Bytes()

		_, err = udpConn.WriteToUDP(response, source)
		if err != nil {
			fmt.Println("Failed to send response:", err)
		}
	}
}
