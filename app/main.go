package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"strings"
)

type header struct {
	ID      uint16
	Flags   uint16
	QDCount uint16
	ANCount uint16
	NSCount uint16
	ARCount uint16
}

type question struct {
	Name  []byte
	QType uint16
	Class uint16
}

type answer struct {
	Name   []byte
	AType  uint16
	Class  uint16
	TTL    uint32
	Length uint16
	Data   []byte
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

func parseHeader(buf []byte) (header, error) {
	h := header{}
	reader := bytes.NewReader(buf)
	err := binary.Read(reader, binary.BigEndian, &h)
	if err != nil {
		return h, err
	}

	flags := h.Flags & (0b01111001 << 8)
	flags |= 1 << 15

	if h.Flags&(0b01111<<11) != 0 {
		fmt.Println("Not a query, ignoring")
		// set response code to 4
		flags |= 4
	}

	h.Flags = flags
	h.QDCount = 1
	h.ANCount = 1
	h.NSCount = 0
	h.ARCount = 0
	return h, nil
}

func parseQuestion(buf []byte) question {
	q := question{}

	for _, v := range buf[12:] {
		q.Name = append(q.Name, v)
		if int(v) == 0 {
			break
		}
	}

	q.QType = 1
	q.Class = 1
	return q
}

func parseAnswer(buf []byte) answer {
	a := answer{}

	for _, v := range buf[12:] {
		a.Name = append(a.Name, v)
		if int(v) == 0 {
			break
		}
	}

	a.AType = 1
	a.Class = 1
	a.TTL = 60
	a.Length = 4
	a.Data = encodeIP("8.8.8.8")
	return a
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

		h, err := parseHeader(buf)
		if err != nil {
			fmt.Println("Failed to parse header:", err)
			break
		}

		q := parseQuestion(buf)
		a := parseAnswer(buf)

		buf := new(bytes.Buffer)
		binary.Write(buf, binary.BigEndian, h)
		binary.Write(buf, binary.BigEndian, q.Name)
		binary.Write(buf, binary.BigEndian, q.QType)
		binary.Write(buf, binary.BigEndian, q.Class)
		binary.Write(buf, binary.BigEndian, a.Name)
		binary.Write(buf, binary.BigEndian, a.AType)
		binary.Write(buf, binary.BigEndian, a.Class)
		binary.Write(buf, binary.BigEndian, a.TTL)
		binary.Write(buf, binary.BigEndian, a.Length)
		binary.Write(buf, binary.BigEndian, a.Data)
		response := buf.Bytes()

		_, err = udpConn.WriteToUDP(response, source)
		if err != nil {
			fmt.Println("Failed to send response:", err)
		}
	}
}
