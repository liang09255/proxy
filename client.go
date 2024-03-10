package main

import (
	"golang.org/x/net/proxy"
	"golang.zx2c4.com/wireguard/tun"
	"golang.zx2c4.com/wireguard/windows/tunnel/winipcfg"
	"io"
	"log"
	"net"
	"net/netip"
	"time"
)

const DefaultMTU = 1420

func main() {
	device := initTun("TestTun", "10.0.0.1/24")
	defer device.Close()
	// 拿到socks5 dialer
	dialer := getSocks5TCPDialer("127.0.0.1:3000")
	// 监听端口接受连接请求
	listener, err := net.Listen("tcp", "10.0.0.1:1000")
	if err != nil {
		panic(err)
	}
	defer listener.Close()
	l := listener.(*net.TCPListener)

	log.Println("Listening on", listener.Addr())
	//持续接收连接请求
	for {
		conn, err := l.AcceptTCP()
		if err != nil {
			log.Println("Error accepting connection:", err)
			continue
		}
		log.Println("获得一个新连接")
		go handleTCPConnect(conn, dialer)
	}
}

func handleTCPConnect(clientConn *net.TCPConn, dialer proxy.Dialer) {
	defer clientConn.Close()
	// 获取到目标ip及端口
	log.Println(clientConn.RemoteAddr().String(), clientConn.LocalAddr().String())
	// 建立tcp连接
	serverConn, err := dialer.Dial("tcp", "10.0.0.1:8889")
	if err != nil {
		log.Println("Error connecting to server:", err)
		return
	}
	defer serverConn.Close()
	// copy
	go io.Copy(serverConn, clientConn)
	io.Copy(clientConn, serverConn)
}

// 初始化tun
func initTun(name, ipStr string) tun.Device {
	// 创建tun
	device, err := tun.CreateTUN(name, DefaultMTU)
	if err != nil {
		panic(err)
	}
	// 绑定ip
	nativeTunDevice := device.(*tun.NativeTun)
	link := winipcfg.LUID(nativeTunDevice.LUID())
	ip, err := netip.ParsePrefix(ipStr)
	if err != nil {
		panic(err)
	}
	err = link.SetIPAddresses([]netip.Prefix{ip})
	if err != nil {
		panic(err)
	}
	return device
}

// 获取socks5 dialer
func getSocks5TCPDialer(addr string) proxy.Dialer {
	dialer, err := proxy.SOCKS5("tcp", addr, nil,
		&net.Dialer{
			Timeout:   3 * time.Second,
			KeepAlive: 30 * time.Second,
		},
	)
	if err != nil {
		panic(err)
	}
	return dialer
}
