package main

import (
	"encoding/binary"
	"fmt"
	"net"
)

var _ = net.ListenUDP

type resultCode int

const (
	noError resultCode = iota
	formErr
	servFail
	nxDomain
	notImp
	refused
)

type dnsHeader struct {
	id uint16 // 16 bits

	recursionDesired    uint8 // 1 bit
	truncatedMessage    uint8 // 1 bit
	authoritativeAnswer uint8 // 1 bit
	opcode              uint8 // 4 bits
	response            uint8 // 1 bit

	rescode            resultCode
	checkingDisabled   uint8
	authedData         uint8
	z                  uint8
	recursionAvailable uint8

	questions            uint16
	answers              uint16
	authoritativeEntries uint16
	resourceEntries      uint16
}

func newHeader() *dnsHeader {
	h := dnsHeader{
		id:       1234,
		response: 1,
	}
	return &h
}

func (h *dnsHeader) toBytes() []byte {
	headerBytes := make([]byte, 12)
	binary.BigEndian.PutUint16(headerBytes[0:2], h.id) // 2 bytes = 16 bits
	headerBytes[2] = h.response << 7                   // shift 7 times to left = 0b10000000 = decimal 128
	return headerBytes
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

		// response := make([]byte, 12)
		response := newHeader().toBytes()

		// header := newHeader()
		// copy(response, buf[:12])
		fmt.Println("header", newHeader())
		fmt.Println("res", response)

		_, err = udpConn.WriteToUDP(response, source)
		if err != nil {
			fmt.Println("Failed to send response:", err)
		}
	}
}
