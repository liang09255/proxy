package main

import (
	"fmt"
	"golang.org/x/net/proxy"
	"net"
	"os"
)

func main() {
	// 监听本地IP数据包
	conn, err := net.ListenPacket("ip", "20.0.0.1:")
	net.FileListener()
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	// 使用SOCKS5代理连接到服务端
	dialer, err := proxy.SOCKS5("tcp", "39.99.236.131:3000", nil, proxy.Direct)
	if err != nil {
		panic(err)
	}

	// 循环接收IP数据包并通过代理发送到服务端
	buf := make([]byte, 1500)
	for {
		n, addr, err := conn.ReadFrom(buf)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading packet: %v\n", err)
			continue
		}

		// 发送数据包到服务端
		proxyConn, err := dialer.Dial("tcp", addr.String())
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error connecting to proxy: %v\n", err)
			continue
		}
		defer proxyConn.Close()

		_, err = proxyConn.Write(buf[:n])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error writing to proxy: %v\n", err)
			continue
		}
	}
}
