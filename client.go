package main

import (
	"fmt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
	"golang.org/x/net/proxy"
	"golang.zx2c4.com/wireguard/tun"
	"golang.zx2c4.com/wireguard/windows/tunnel/winipcfg"
	"log"
	"net/netip"
	"regexp"
	"sync"
)

const DefaultMTU2 = 1420
const (
	ProtocolICMP = 1
	ProtocolTCP  = 6
	ProtocolUDP  = 17
)

var tun2 tun.Device

func main() {
	tun2 = initTun2("Test Tun", "20.0.0.1/24")
	for {
		// 第一个字节获取到协议版本号和头部长度
		buf := make([]byte, 2048)
		n, err := tun2.Read(buf, 0)
		buf = buf[:n]
		if err != nil {
			panic(err)
		}
		if n >= 2047 {
			log.Println("超级长")
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
				if tcp.SYN {
					err = firstHandshake(ip, tcp)
					if err != nil {
						log.Println(err)
						continue
					}
				} else if tcp.ACK && !tcp.PSH {
					log.Println("第三次握手", tcp.ACK, tcp.Seq)
				} else if tcp.ACK && tcp.PSH {
					log.Println("开始推送数据")
					err = proxyPacket(ip, tcp)
					if err != nil {
						log.Println(err)
						continue
					}
				}
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

func firstHandshake(ip *layers.IPv4, tcp *layers.TCP) (err error) {
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
		FixLengths:       true,
		ComputeChecksums: true,
	}
	err = tcp2.SetNetworkLayerForChecksum(ip)
	if err != nil {
		return err
	}
	buffer := gopacket.NewSerializeBuffer()
	err = gopacket.SerializeLayers(buffer, options, ip2, tcp2)
	if err != nil {
		return fmt.Errorf("error serializing packet: %s", err.Error())
	}
	_, err = tun2.Write(buffer.Bytes(), 0)
	log.Println(buffer.Bytes())
	return err
}

func proxyPacket(ip *layers.IPv4, tcp *layers.TCP) (err error) {
	dialer := getSocks5TCPDialer2()
	re := regexp.MustCompile(`(\d+)`) // 匹配数字
	match := re.FindStringSubmatch(tcp.DstPort.String())
	addr := ip.DstIP.String() + ":" + match[1]
	conn, err := dialer.Dial("tcp", addr)
	if err != nil {
		return err
	}
	log.Println("代理连接成功")

	resp := make([]byte, 2048)
	wg := sync.WaitGroup{}
	wg.Add(2)
	log.Println("传输数据ing")
	go func() {
		defer wg.Done()
		n1, err := conn.Write(tcp.Payload)
		if err != nil {
			log.Println(err)
		}
		log.Println("发送了", n1)
	}()
	go func() {
		defer wg.Done()
		n2, err := conn.Read(resp)
		if err != nil {
			log.Println(err)
		}
		log.Println("接收到", n2)
		resp = resp[:n2]
	}()
	wg.Wait()
	log.Println("传输完成")
	// 封装成ip包写回去
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
		PSH:     true,
		ACK:     true,
		Ack:     tcp.Seq + uint32(len(resp)),
		Seq:     tcp.Ack,
		Window:  65535,
	}
	options := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}
	err = tcp2.SetNetworkLayerForChecksum(ip)
	if err != nil {
		return err
	}
	buffer := gopacket.NewSerializeBuffer()
	err = gopacket.SerializeLayers(buffer, options, ip2, tcp2, gopacket.Payload(resp))
	if err != nil {
		return err
	}
	_, err = tun2.Write(buffer.Bytes(), 0)
	log.Println(buffer.Bytes())
	return err
}
