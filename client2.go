package main

import (
	"encoding/binary"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
	"golang.org/x/net/proxy"
	"golang.zx2c4.com/wireguard/tun"
	"golang.zx2c4.com/wireguard/windows/tunnel/winipcfg"
	"log"
	"net"
	"net/netip"
	"strconv"
	"time"
)

const DefaultMTU2 = 1420
const (
	ProtocolICMP = 1
	ProtocolTCP  = 6
	ProtocolUDP  = 17
)

func main() {
	tun2 := initTun2("Test Tun", "20.0.0.1/24")
	for {
		// 第一个字节获取到协议版本号和头部长度
		buf := make([]byte, 1024)
		_, err := tun2.Read(buf, 0)
		if err != nil {
			panic(err)
		}
		switch buf[0] >> 4 {
		case ipv4.Version:
			//headerLength := (buf[0] & 0x0F) * 4
			//buf = make([]byte, headerLength)
			//_, err = tun2.Read(buf, 0)
			//if err != nil {
			//	log.Println(err)
			//	continue
			//}
			header, err := ipv4.ParseHeader(buf)
			if err != nil {
				log.Println(err)
				continue
			}
			if header.Dst.String() != "39.99.236.131" {
				continue
			}
			log.Println("接收到一个ipv4数据包", "源地址:", header.Src, "目的地址:", header.Dst)
			if header.TotalLen == 0 {
				continue
			}
			data := buf[header.Len:]
			switch header.Protocol {
			case ProtocolICMP:
				log.Println("暂不支持ICMP协议")
			case ProtocolTCP:
				sourcePort := binary.BigEndian.Uint16(data[0:2])
				dstPort := binary.BigEndian.Uint16(data[2:4])
				log.Println("源端口:", sourcePort, "目的端口:", dstPort)
				dialer := getSocks5TCPDialer2()
				conn, err := dialer.Dial("tcp", header.Dst.String()+":"+strconv.FormatInt(int64(dstPort), 10))
				if err != nil {
					log.Println(err)
					continue
				}
				// TODO 转发数据

			case ProtocolUDP:
			default:
			}
		case ipv6.Version:
		default:
		}
	}
}

// 初始化tun
func initTun2(name, ipStr string) tun.Device {
	// 创建tun
	device, err := tun.CreateTUN(name, DefaultMTU2)
	if err != nil {
		panic(err)
	}
	// 绑定ip
	ip, err := netip.ParsePrefix(ipStr)
	if err != nil {
		panic(err)
	}
	err = winipcfg.LUID(device.(*tun.NativeTun).LUID()).SetIPAddresses([]netip.Prefix{ip})
	if err != nil {
		panic(err)
	}
	return device
}

func getSocks5TCPDialer2() proxy.Dialer {
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
	return dialer
}
