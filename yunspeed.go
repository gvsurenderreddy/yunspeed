package main

import (
	"bufio"
	"bytes"
	"container/list"
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"strings"
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
	Host            string
	Count           int32
	DurationList    *list.List
	SendedPackets   int32
	min             int32
	max             int32
	avg             float32
	lostPacketsRate float32
}

func (s *StasticData) PrintStasticData() {
	var min, max, sum int32
	if s.DurationList.Len() == 0 {
		min, max, sum = 0, 0, 0
	} else {
		min, max, sum = s.DurationList.Front().Value.(int32), s.DurationList.Front().Value.(int32), int32(0)
	}

	for ele := s.DurationList.Front(); ele != nil; ele = ele.Next() {
		value := ele.Value.(int32)
		if max < value {
			max = value
		}

		if min > value {
			min = value
		}

		sum += value
	}

	recvdPackets, lostPackets := s.DurationList.Len(), s.SendedPackets-int32(s.DurationList.Len())

	s.min = min
	s.max = max
	if recvdPackets == 0 {
		s.avg = float32(0)
	} else {
		s.avg = float32(sum) / float32(recvdPackets)
	}
	s.lostPacketsRate = float32(lostPackets) / float32(s.SendedPackets) * 100

	fmt.Printf("%d packets transmitted, %d received, %.1f%% packet loss\n rtt min/avg/max = %d/%.1f/%d\n",
		s.SendedPackets, recvdPackets, s.lostPacketsRate,
		min, s.avg, max)
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
	result.Host = host
	result.Count = int32(count)
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

		//read timeout
		//TODO: could be configurable
		conn.SetReadDeadline(tNow.Add(1 * time.Second))
		recv := make([]byte, 512)
		_, err = conn.Read(recv)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error : %s\n", err.Error())
			continue
		}

		tEnd := time.Now()

		duration := int32(tEnd.Sub(tNow).Nanoseconds() / (1000 * 1000))
		fmt.Printf("from %s : time=%dms\n", addr.String(), duration)
		result.DurationList.PushBack(duration)

		// fmt.Println(recv)
	}
	return result

}

func main() {

	argsLen := len(os.Args)
	if argsLen < 2 {
		fmt.Print(
			"Please run with [super user] in terminal\n",
			"Usage:\n",
			"yunspeed hostListFilePath\n",
			"for example: yunspeed /home/yuntivpns.txt\n")
		os.Exit(1)
	}
	count := 10

	hostFile := os.Args[1]

	hosts, err := readHostFile(hostFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Read file error : %s", err.Error())
	}

	// fmt.Printf("hosts length : %d\n", len(hosts))

	var chs []chan StasticData

	for i := 0; i < len(hosts); i++ {
		chs = append(chs, make(chan StasticData))
		go pingHost(hosts[i], count, chs[i])
	}

	var datas []StasticData
	for _, ch := range chs {
		data := <-ch
		// fmt.Printf("Result From : %s \n", data.Host)
		datas = append(datas, data)
	}

	// fmt.Printf("result length : %d\n", len(datas))

	timeThreshold := 200
	allBelowThreshold := true
	recommendIndex := 0
	minTtl := datas[0].avg

	for index := 0; index < len(datas); index++ {
		data := datas[index]
		if data.avg > float32(timeThreshold) || data.avg < 1 {
			continue
		}

		if allBelowThreshold {
			minTtl = data.avg
		}
		allBelowThreshold = false
		if minTtl > data.avg {
			minTtl = data.avg
			recommendIndex = index
		}

	}

	// fmt.Printf("recommend index : %d\n", recommendIndex)

	if allBelowThreshold {
		fmt.Println("no recommend")
	} else {
		fmt.Printf("Recommend Host : %s (rtt min/avg/max = %d/%.1f/%d)\n",
			datas[recommendIndex].Host, datas[recommendIndex].min, datas[recommendIndex].avg, datas[recommendIndex].max)
	}

	os.Exit(0)
}

func readHostFile(filePath string) ([]string, error) {
	var hosts []string
	file, err := os.Open(filePath)
	if err != nil {
		return hosts, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		hosts = append(hosts, strings.TrimSpace(scanner.Text()))
	}
	return hosts, scanner.Err()

}

func pingHost(host string, count int, ch chan StasticData) {
	stasticData := sendICMPPacket(host, count)

	stasticData.PrintStasticData()

	ch <- stasticData

	fmt.Println("Done", stasticData.Host)
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
