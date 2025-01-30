package main

import (
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"net"
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
	h.ANCount = h.QDCount
	h.NSCount = 0
	h.ARCount = 0
	return h, nil
}

func parseQuestion(buf []byte, numQuestions uint16) []question {
	questions := []question{}
	offset := 12 // inital offset = header bytes
	i := 0
	q := question{}
	for _, v := range buf[offset:] {
		q.Name = append(q.Name, v)
		if int(v) == 0 {
			q.QType = 1
			q.Class = 1
			questions = append(questions, q)
			i++
			offset += 4 + 1 // 2 bytes for type, 2 bytes for class, next
			q = question{}
		}
		if i == int(numQuestions) {
			break
		}
	}
	return questions
}

func parseAnswer(buf []byte) answer {
	answer := answer{}
	for i, v := range buf {
		answer.Name = append(answer.Name, v)
		if int(v) == 0 {
			answer.AType = binary.BigEndian.Uint16(buf[i+1 : i+3])
			answer.Class = binary.BigEndian.Uint16(buf[i+3 : i+5])
			answer.TTL = binary.BigEndian.Uint32(buf[i+5 : i+9])
			answer.Length = binary.BigEndian.Uint16(buf[i+9 : i+11])
			answer.Data = buf[i+11:]
			break
		}
	}
	return answer
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

	forwardAddress := flag.String("resolver", "", "address to forward questions")
	flag.Parse()

	var forwardAddr *net.UDPAddr
	if forwardAddress != nil {
		forwardAddr, err = net.ResolveUDPAddr("udp", *forwardAddress)
		if err != nil {
			fmt.Println("Failed to resolve forward UDP address:", err)
			return
		}
	}

	requestBuf := make([]byte, 512)

	for {
		size, source, err := udpConn.ReadFromUDP(requestBuf)
		if err != nil {
			fmt.Println("Error receiving data:", err)
			break
		}

		receivedData := string(requestBuf[:size])
		fmt.Printf("Received %d bytes from %s: %s\n", size, source, receivedData)

		h, err := parseHeader(requestBuf)
		if err != nil {
			fmt.Println("Failed to parse header:", err)
			break
		}

		questions := parseQuestion(requestBuf, h.QDCount)

		responseBuf := new(bytes.Buffer)
		binary.Write(responseBuf, binary.BigEndian, h)

		for _, q := range questions {
			binary.Write(responseBuf, binary.BigEndian, q.Name)
			binary.Write(responseBuf, binary.BigEndian, q.QType)
			binary.Write(responseBuf, binary.BigEndian, q.Class)
		}

		for _, q := range questions {
			forwardBuf := make([]byte, 512)
			qBuf := new(bytes.Buffer)
			binary.Write(qBuf, binary.BigEndian, q.Name)
			binary.Write(qBuf, binary.BigEndian, q.QType)
			binary.Write(qBuf, binary.BigEndian, q.Class)

			dataWithSingleQuestion := []byte{}
			dataWithSingleQuestion = append(dataWithSingleQuestion, requestBuf[:12]...) // request header
			dataWithSingleQuestion = append(dataWithSingleQuestion, qBuf.Bytes()...)
			_, err = udpConn.WriteToUDP(dataWithSingleQuestion, forwardAddr)
			if err != nil {
				fmt.Println("Failed to forward request:", err)
				return
			}

			_, _, err := udpConn.ReadFromUDP(forwardBuf)
			if err != nil {
				fmt.Println("Error receiving data from foward:", err)
				return
			}

			a := parseAnswer(forwardBuf[12+len(qBuf.Bytes()):])
			binary.Write(responseBuf, binary.BigEndian, a.Name)
			binary.Write(responseBuf, binary.BigEndian, a.AType)
			binary.Write(responseBuf, binary.BigEndian, a.Class)
			binary.Write(responseBuf, binary.BigEndian, a.TTL)
			binary.Write(responseBuf, binary.BigEndian, a.Length)
			binary.Write(responseBuf, binary.BigEndian, a.Data)
		}

		response := responseBuf.Bytes()

		_, err = udpConn.WriteToUDP(response, source)
		if err != nil {
			fmt.Println("Failed to send response:", err)
		}
	}
}
