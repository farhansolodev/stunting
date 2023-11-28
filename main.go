package main

import (
	// "bufio"
	"bufio"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strings"

	// "os"
	// "strings"
	// "time"

	"github.com/pion/stun/v2"
)

var server = flag.String("server", "94.130.130.49:3478", "Stun server address")

const (
	udp           = "udp4"
	pingMsg       = "ping"
	pongMsg       = "pong"
	timeoutMillis = 500
)

func main() {}

func getPeerAddr() string {
	reader := bufio.NewReader(os.Stdin)
	log.Println("Enter remote peer address:")
	peer, _ := reader.ReadString('\n')
	return strings.Trim(peer, " \r\n")
}

func listen(conn *net.UDPConn) <-chan []byte {
	messages := make(chan []byte)
	go func() {
		for {
			buf := make([]byte, 1024)
			n, _, err := conn.ReadFromUDP(buf)
			if err != nil {
				close(messages)
				return
			}
			buf = buf[:n]

			messages <- buf
		}
	}()
	return messages
}

func sendBindingRequest(conn *net.UDPConn, addr *net.UDPAddr) error {
	m := stun.MustBuild(stun.TransactionID, stun.BindingRequest)

	err := send(m.Raw, conn, addr)
	if err != nil {
		return fmt.Errorf("binding: %w", err)
	}

	return nil
}

func send(msg []byte, conn *net.UDPConn, addr *net.UDPAddr) error {
	_, err := conn.WriteToUDP(msg, addr)
	if err != nil {
		return fmt.Errorf("send: %w", err)
	}

	return nil
}

func sendStr(msg string, conn *net.UDPConn, addr *net.UDPAddr) error {
	return send([]byte(msg), conn, addr)
}
