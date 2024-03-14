package main

import (
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
	"golang.org/x/net/proxy"
	"golang.zx2c4.com/wireguard/tun"
	"golang.zx2c4.com/wireguard/windows/tunnel/winipcfg"
	"log"
	"net/netip"
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
		buf := make([]byte, 2048)
		n, err := tun2.Read(buf, 0)
		buf = buf[:n]
		if err != nil {
			panic(err)
		}
		switch buf[0] >> 4 {
		case ipv4.Version:
			ipP := gopacket.NewPacket(buf, layers.LayerTypeIPv4, gopacket.Default)
			ipL := ipP.NetworkLayer()
			ip, _ := ipL.(*layers.IPv4)
			if ip.DstIP.String() != "39.108.81.214" {
				continue
			}
			log.Println("接收到一个ipv4数据包", "源地址:", ip.SrcIP, "目的地址:", ip.DstIP)
			if ip.Length == 0 {
				continue
			}

			switch ip.Protocol {
			case ProtocolICMP:
				log.Println("暂不支持ICMP协议")
			case ProtocolTCP:
				tcpP := gopacket.NewPacket(ip.Payload, layers.LayerTypeTCP, gopacket.Default)
				tl := tcpP.TransportLayer()
				tcp, _ := tl.(*layers.TCP)
				log.Println("源端口:", tcp.SrcPort, "目的端口:", tcp.DstPort)
				if tcp.SYN && tcp.Seq != 0 {
					log.Println("第一次握手", tcp.Seq)
					log.Println("第二次握手")

					ip2 := &layers.IPv4{
						Version:  4,
						TTL:      255,
						Flags:    layers.IPv4DontFragment,
						Protocol: layers.IPProtocolTCP,
						SrcIP:    ip.DstIP,
						DstIP:    ip.SrcIP,
					}

					tcp2 := &layers.TCP{
						SrcPort: tcp.DstPort,
						DstPort: tcp.SrcPort,
						SYN:     true,
						ACK:     true,
						Ack:     tcp.Seq + 1,
						Seq:     123,
						Window:  65535,
					}
					options := gopacket.SerializeOptions{
						FixLengths: true,
					}
					buffer := gopacket.NewSerializeBuffer()
					err := gopacket.SerializeLayers(buffer, options, ip2, tcp2)
					if err != nil {
						log.Println("Error serializing packet:", err)
						continue
					}
					n2, err := tun2.Write(buffer.Bytes(), 0)
					log.Println(buffer.Bytes())
					if err != nil {
						log.Println(err)
						continue
					}
					log.Println(n2)
				} else if tcp.ACK && tcp.Ack != 0 && tcp.Seq != 0 {
					log.Println("第三次握手", tcp.ACK, tcp.Seq)
				} else {
					log.Println("开始发送数据")
				}
				//if tcp.SYN && !tcp.ACK {
				//	log.Println("完成连接")
				//	sourcePort := tl.TransportFlow().Src()
				//	dstPort := tl.TransportFlow().Dst()
				//	log.Println("源端口:", sourcePort, "目的端口:", dstPort)
				//	dialer := getSocks5TCPDialer2()
				//	_, err := dialer.Dial("tcp", header.Dst.String()+":"+dstPort.String())
				//	if err != nil {
				//		log.Println(err)
				//		continue
				//	}
				//	al := tcpP.ApplicationLayer()
				//	if al != nil {
				//		log.Println(al.Payload())
				//		log.Println(al.LayerPayload())
				//		log.Println(al.LayerContents())
				//	}
				//}

				//// TODO 转发数据
				//_, err = conn.Write(data)
				//if err != nil {
				//	log.Println(err)
				//	continue
				//}
				////log.Println(string(data))
				//result := make([]byte, 1024)
				//n2, err := conn.Read(result)
				//if err != nil {
				//	log.Println(err)
				//	continue
				//}
				//result = result[:n2]
				//log.Println(string(result))
				//n3, err := tun2.Write(result, 0)
				//if err != nil {
				//	log.Println(err)
				//	continue
				//}
				//log.Println(n3)
				//log.Println(string(result))

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
	addr := "39.99.236.131:3000"
	dialer, err := proxy.SOCKS5("tcp", addr, nil, nil)
	if err != nil {
		panic(err)
	}
	return dialer
}
