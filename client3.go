package main

import (
	"fmt"
	"golang.org/x/net/proxy"
	"io"
	"net"
	"time"
)

func main() {
	addr := "127.0.0.1:3000"
	dialer, err := proxy.SOCKS5("tcp", addr, nil,
		&net.Dialer{
			Timeout:   3 * time.Second,
			KeepAlive: 30 * time.Second,
		},
	)
	if err != nil {
		panic(err)
	}
	// Listen for incoming connections on a specific port
	listener, err := net.Listen("tcp", "0.0.0.0:1000")
	if err != nil {
		fmt.Println("Failed to listen:", err)
		return
	}
	defer listener.Close()

	fmt.Println("Listening on", listener.Addr())

	// Accept incoming connections and handle them
	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}

		go handleConnect(conn, dialer)
	}
}

func handleConnect(clientConn net.Conn, dialer proxy.Dialer) {
	defer clientConn.Close()

	// Connect to the destination server using the dialer
	serverConn, err := dialer.Dial("tcp", "172.31.64.1:8889")
	if err != nil {
		fmt.Println("Error connecting to server:", err)
		return
	}
	defer serverConn.Close()

	go io.Copy(serverConn, clientConn)
	// Forward data from server to client
	io.Copy(clientConn, serverConn)
}
