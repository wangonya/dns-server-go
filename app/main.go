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

type answer struct {
	name   []byte
	atype  uint16
	class  uint16
	ttl    uint32
	length uint16
	data   []byte
}

func encodeDomain(domain string) []byte {
	var encodedDomain []byte
	for _, part := range strings.Split(domain, ".") {
		encodedDomain = append(encodedDomain, byte(len(part)))
		encodedDomain = append(encodedDomain, []byte(part)...)
	}
	encodedDomain = append(encodedDomain, 0)
	return encodedDomain
}

func encodeIP(ip string) []byte {
	var encodedIp []byte
	for _, part := range strings.Split(ip, ".") {
		encodedIp = append(encodedIp, []byte(part)...)
	}
	return encodedIp
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
			ancount: 1,
			nscount: 0,
			arcount: 0,
		}
		q := question{
			name:  encodeDomain("codecrafters.io"),
			qtype: 1,
			class: 1,
		}
		a := answer{
			name:   encodeDomain("codecrafters.io"),
			atype:  1,
			class:  1,
			ttl:    60,
			length: 4,
			data:   encodeIP("8.8.8.8"),
		}

		buf := new(bytes.Buffer)
		binary.Write(buf, binary.BigEndian, h)
		binary.Write(buf, binary.BigEndian, q.name)
		binary.Write(buf, binary.BigEndian, q.qtype)
		binary.Write(buf, binary.BigEndian, q.class)
		binary.Write(buf, binary.BigEndian, a.name)
		binary.Write(buf, binary.BigEndian, a.atype)
		binary.Write(buf, binary.BigEndian, a.class)
		binary.Write(buf, binary.BigEndian, a.ttl)
		binary.Write(buf, binary.BigEndian, a.length)
		binary.Write(buf, binary.BigEndian, a.data)
		response := buf.Bytes()

		_, err = udpConn.WriteToUDP(response, source)
		if err != nil {
			fmt.Println("Failed to send response:", err)
		}
	}
}
