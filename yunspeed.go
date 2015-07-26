package main

import (
	"fmt"
	"net"
	"os"
)

func main() {

	host := "www.baidu.com"

	addr, err := net.ResolveIPAddr("ip", host)
	if err != nil {
		fmt.Println("Resolve Address Error", err.Error())
		os.Exit(1)
	}
	laddr := net.IPAddr{IP: net.ParseIP("0.0.0.0")}

	fmt.Printf("%s ->  %s", host, addr.IP.To16())
	fmt.Println()

	conn, err := net.DialIP("ip4:icmp", &laddr, addr)
	checkErr(err)
	defer conn.Close()

	fmt.Println("Dial successfully")

	var msg [512]byte
	msg[0] = 8  //echo
	msg[1] = 0  //code
	msg[2] = 0  //checksum
	msg[3] = 0  //checksum
	msg[4] = 0  //identifier[0]
	msg[5] = 13 //identifier[1]
	msg[6] = 0  //sequence[0]
	msg[7] = 40 //sequence[1]
	len := 8

	checksum := checksum(msg[0:len])
	msg[2] = byte(checksum >> 8)
	msg[3] = byte(checksum & 255)

	_, err = conn.Write(msg[0:len])
	checkErr(err)
	fmt.Println("Send icmp packet successfully")

	_, err = conn.Read(msg[:])
	checkErr(err)

	fmt.Println("Got Response")

	fmt.Println(msg)

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
