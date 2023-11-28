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
	"time"

	"github.com/pion/stun/v2"
)

var server = flag.String("server", "94.130.130.49:3478", "Stun server address")

const (
	udp           = "udp4"
	pingMsg       = "ping"
	pongMsg       = "pong"
	timeoutMillis = 500
)

// -a test
func main() { // nolint:gocognit
	flag.Parse()

	// allocate a local UDP socket
	conn, err := net.ListenUDP(udp, &net.UDPAddr{
		Port: 50002,
	})
	if err != nil {
		log.Fatalf("Failed to listen: %s", err)
	}
	defer conn.Close()

	// listen for incoming messages on the socket
	messageChan := listen(conn)
	log.Printf("Listening on %s", conn.LocalAddr())

	// get *net.UDPAddr of STUN server address string
	srvAddr, err := net.ResolveUDPAddr(udp, *server)
	if err != nil {
		log.Panicln("Failed to resolve server addr:", err)
	}
	// send binding request to STUN server
	if sendBindingRequest(conn, srvAddr) != nil {
		log.Panicln("sendBindingRequest:", err)
	}

	var publicAddr stun.XORMappedAddress
	var peerAddr *net.UDPAddr

	gotPong := false
	sentPong := false

	// start sending messages to this channel every few ms
	keepalive := time.Tick(timeoutMillis * time.Millisecond)
	keepaliveMsg := pingMsg

	for {
		// if sent and received the same "pong" message,
		if gotPong && sentPong {
			log.Println("Success! Quitting.")
			conn.Close()
		}

		select {
		case message, ok := <-messageChan:
			if !ok {
				return
			}

			switch {
			case string(message) == pingMsg:
				keepaliveMsg = pongMsg

			case string(message) == pongMsg:
				if !gotPong {
					log.Println("Received pong message.")
				}
				// One client may skip sending ping if it receives
				// a ping message before knowning the peer address.
				keepaliveMsg = pongMsg
				gotPong = true

			case stun.IsMessage(message):
				m := new(stun.Message)
				m.Raw = message
				decErr := m.Decode()
				if decErr != nil {
					log.Println("decode:", decErr)
					break
				}
				var xorAddr stun.XORMappedAddress
				if getErr := xorAddr.GetFrom(m); getErr != nil {
					log.Println("getFrom:", getErr)
					break
				}

				if publicAddr.String() != xorAddr.String() {
					log.Printf("My public address: %s\n", xorAddr)
					publicAddr = xorAddr

					if peerAddr == nil {
						// get peer address from user and set it
						addr, err := net.ResolveUDPAddr(udp, getPeerAddr())
						if err != nil {
							log.Panicln("resolve peeraddr:", err)
						}
						peerAddr = addr
					}
				}

			default:
				log.Panicln("unknown message", message)
			}

		// case peerStr := <-peerAddrChan:
		// 	peerAddr, err = net.ResolveUDPAddr(udp, peerStr)
		// 	if err != nil {
		// 		log.Panicln("resolve peeraddr:", err)
		// 	}

		case <-keepalive:
			if peerAddr == nil {
				continue
			}
			// Keep NAT binding alive using the peer address
			err = sendStr(keepaliveMsg, conn, peerAddr)
			if keepaliveMsg == pongMsg {
				sentPong = true
			}

			if err != nil {
				log.Panicln("keepalive:", err)
			}
		}
	}
}

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
