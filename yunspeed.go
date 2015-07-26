package main

import (
	"bytes"
	"container/list"
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"time"
)

type ICMP struct {
	Type       uint8
	Code       uint8
	Checksum   uint16
	Identifier uint16
	Sequence   uint16
}

type StasticData struct {
	Count         int
	DurationList  *list.List
	SendedPackets int
}

func (s *StasticData) PrintStasticData() {
	var min, max, sum int64
	if s.DurationList.Len() == 0 {
		min, max, sum = 0, 0, 0
	} else {
		min, max, sum = s.DurationList.Front().Value.(int64), s.DurationList.Front().Value.(int64), int64(0)
	}

	for ele := s.DurationList.Front(); ele != nil; ele = ele.Next() {
		value := ele.Value.(int64)
		if max < value {
			max = value
		}

		if min > value {
			min = value
		}

		sum += value
	}

	recvdPackets, lostPackets := s.DurationList.Len(), s.SendedPackets-s.DurationList.Len()
	fmt.Printf("%d packets transmitted, %d received, %.1f%% packet loss\n rtt min/avg/max = %d/%.1f/%d\n",
		s.SendedPackets, recvdPackets, float32(lostPackets)/float32(s.SendedPackets)*100,
		min, float32(sum)/float32(recvdPackets), max)
}

func sendICMPPacket(host string, count int) StasticData {
	addr, err := net.ResolveIPAddr("ip", host)
	if err != nil {
		fmt.Println("Resolve Address Error", err.Error())
		os.Exit(1)
	}
	laddr := net.IPAddr{IP: net.ParseIP("0.0.0.0")}

	conn, err := net.DialIP("ip4:icmp", &laddr, addr)
	checkErr(err)
	defer conn.Close()

	var icmp ICMP
	icmp.Type = 8
	icmp.Code = 0
	icmp.Checksum = 0
	icmp.Identifier = 13
	icmp.Sequence = 40

	var buffer bytes.Buffer
	binary.Write(&buffer, binary.BigEndian, icmp)
	icmp.Checksum = checksum(buffer.Bytes())
	buffer.Reset()
	binary.Write(&buffer, binary.BigEndian, icmp)

	var result StasticData
	result.Count = count
	result.DurationList = list.New()

	fmt.Printf("\nPing %s (%s) 0 bytes of data\n", host, addr.IP.To16())

	for i := 0; i < count; i++ {
		_, err = conn.Write(buffer.Bytes())
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error : %s\n", err.Error())
			continue
		}
		tNow := time.Now()
		result.SendedPackets++
		// fmt.Println("Send icmp packet successfully")

		conn.SetReadDeadline(tNow.Add(5 * time.Second))
		recv := make([]byte, 512)
		_, err = conn.Read(recv)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error : %s\n", err.Error())
			continue
		}

		tEnd := time.Now()

		duration := (tEnd.Sub(tNow).Nanoseconds() / (1000 * 1000))
		fmt.Printf("from %s : time=%dms\n", addr.String(), duration)
		result.DurationList.PushBack(duration)

		// fmt.Println(recv)
	}
	return result

}

func main() {

	host := "us1.vpnfax.com"
	count := 4

	stasticData := sendICMPPacket(host, count)

	stasticData.PrintStasticData()

	os.Exit(0)
}

func checkErr(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal Error : %s", err.Error())
		os.Exit(1)
	}
}

func checksum(msg []byte) uint16 {
	sum := 0

	for n := 0; n < len(msg); n += 2 {
		sum += int(msg[n])*256 + int(msg[n+1])
	}
	var answer uint16 = uint16(^sum)
	return answer
}
