package main

import (
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
)

func main() {
	proxyAddr := "127.0.0.1:7070"
	targetAddr := "127.0.0.1:8080"

	conn, err := dialSocksProxy(proxyAddr, targetAddr)
	if err != nil {
		fmt.Println("Error connecting via SOCKS5 proxy:", err)
		return
	}
	defer conn.Close()

	// 使用 conn 进行通信...
	// 构建 HTTP 请求
	request := "POST / HTTP/1.1\n" +
		"Host: LxsTest\n" +
		"User-Agent: LXS Custom Client\n" +
		"Connection: close\n\n"
	// 发送请求
	_, err = conn.Write([]byte(request))
	if err != nil {
		fmt.Println("Error sending HTTP GET request:", err)
		return
	}

	// 读取响应头部
	var responseHeader string
	buffer := make([]byte, 1024)
	for {
		n, err := conn.Read(buffer)
		if err != nil {
			fmt.Println("Error reading HTTP response:", err)
			return
		}
		responseHeader += string(buffer[:n])
		if strings.Contains(responseHeader, "\r\n\r\n") {
			break
		}
	}

	// 获取主体起始位置
	bodyStart := strings.Index(responseHeader, "\r\n\r\n") + 4

	// 读取主体内容
	var responseBody string
	responseBody += responseHeader[bodyStart:]
	for {
		n, err := conn.Read(buffer)
		if err != nil {
			if err != io.EOF {
				fmt.Println("Error reading HTTP response body:", err)
			}
			break
		}
		responseBody += string(buffer[:n])
	}

	fmt.Println("Response Body:", responseBody)
}

func dialSocksProxy(proxyAddr, targetAddr string) (net.Conn, error) {
	proxyConn, err := net.Dial("tcp", proxyAddr)
	if err != nil {
		return nil, err
	}

	// 发送 SOCKS5 协商请求
	_, err = proxyConn.Write([]byte{0x05, 0x01, 0x00})
	if err != nil {
		return nil, err
	}

	// 读取 SOCKS5 协商响应
	response := make([]byte, 2)
	_, err = proxyConn.Read(response)
	if err != nil {
		return nil, err
	}
	if response[0] != 0x05 || response[1] != 0x00 {
		return nil, errors.New("SOCKS5 negotiation failed")
	}

	// 发送连接请求
	targetHost, targetPort, _ := net.SplitHostPort(targetAddr)
	targetPortInt, _ := strconv.Atoi(targetPort)
	targetPortBytes := []byte{byte(targetPortInt >> 8), byte(targetPortInt & 0xFF)}

	req := []byte{0x05, 0x01, 0x00, 0x03, byte(len(targetHost))}
	req = append(req, []byte(targetHost)...)
	req = append(req, targetPortBytes...)
	_, err = proxyConn.Write(req)
	if err != nil {
		return nil, err
	}

	// 读取连接响应
	response = make([]byte, 10)
	_, err = proxyConn.Read(response)
	if err != nil {
		return nil, err
	}
	if response[1] != 0x00 {
		return nil, errors.New("SOCKS5 connection failed")
	}

	return proxyConn, nil
}
