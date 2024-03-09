package main

import (
	"encoding/binary"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
	"golang.org/x/net/proxy"
	"golang.zx2c4.com/wireguard/tun"
	"golang.zx2c4.com/wireguard/windows/tunnel/winipcfg"
	"io"
	"log"
	"net"
	"net/netip"
	"strconv"
	"time"
)

const (
	ProtocolICMP = 1
	ProtocolTCP  = 6
	ProtocolUDP  = 17
)

func main() {
	ifname := "MyNIC"
	dev, err := tun.CreateTUN(ifname, 0)
	if err != nil {
		panic(err)
	}
	defer dev.Close()
	// 保存原始设备句柄
	nativeTunDevice := dev.(*tun.NativeTun)

	// 获取LUID用于配置网络
	link := winipcfg.LUID(nativeTunDevice.LUID())

	ip, err := netip.ParsePrefix("10.0.0.1/24")
	if err != nil {
		panic(err)
	}
	err = link.SetIPAddresses([]netip.Prefix{ip})
	if err != nil {
		panic(err)
	}

	batchSize := dev.BatchSize()

	n := 2048
	bufs := make([][]byte, batchSize)
	for i := range bufs {
		bufs[i] = make([]byte, n)
	}

	addr := "127.0.0.1:3000"
	dialer, err := proxy.SOCKS5("tcp", addr, nil,
		&net.Dialer{
			Timeout:   3 * time.Second,
			KeepAlive: 30 * time.Second,
		},
	)

	// 读取ICMP
	for {
		size := make([]int, batchSize)
		n, err = dev.Read(bufs, size, 0)
		if err != nil {
			panic(err)
		}
		for i := 0; i < n; i++ {
			if size[i] < 1 {
				continue
			}
			buf := bufs[i][:size[i]]
			// 解析IP头 ipv4:0010xxxx
			switch buf[0] >> 4 {
			case ipv4.Version:
				if len(buf) < ipv4.HeaderLen {
					continue
				}
				header, err := ipv4.ParseHeader(buf)
				if err != nil {
					log.Println(err)
					continue
				}
				log.Println("Src:", header.Src, " dst:", header.Dst)
				data := buf[header.Len:]

				switch header.Protocol {
				case ProtocolICMP:
					msg, _ := icmp.ParseMessage(ProtocolICMP, data)
					log.Println(">> ICMP:", msg.Type)
				case ProtocolTCP:
					sourcePort := binary.BigEndian.Uint16(data[0:2])
					dstPort := binary.BigEndian.Uint16(data[2:4])
					log.Println(">> TCP:", sourcePort, dstPort)
					conn, err := dialer.Dial("tcp", header.Dst.String()+":"+strconv.FormatInt(int64(dstPort), 10))
					if err != nil {
						panic(err)
					}
					defer conn.Close()

					num, err := conn.Write(buf)
					if err != nil {
						panic(err)
					}
					log.Println(">> TCP:", num)
					io.Copy(conn, dev.File())
					res, err := io.ReadAll(conn)
					log.Println("<< TCP:", string(res))

				case ProtocolUDP:
				default:
					log.Println("not support ipv4 protocol:", header.Protocol)
				}
			case ipv6.Version:
				if len(buf) < ipv6.HeaderLen {
					continue
				}
				header, err := ipv6.ParseHeader(buf)
				if err != nil {
					log.Println(err)
					continue
				}
				log.Println("Ipv6 packet not supported", "Src:", header.Src, " dst:", header.Dst)
			default:
				log.Println("Unknown protocol")
			}
		}
	}
}
