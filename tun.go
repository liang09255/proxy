package main

import (
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.zx2c4.com/wireguard/tun"
	"golang.zx2c4.com/wireguard/windows/tunnel/winipcfg"
	"log"
	"net/netip"
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

	ip, err := netip.ParsePrefix("10.0.0.77/24")
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

	// 读取ICMP
	for {
		size := make([]int, batchSize)
		n, err = dev.Read(bufs, size, 0)
		if err != nil {
			panic(err)
		}
		for i := 0; i < n; i++ {
			buf := bufs[i][:size[i]]
			header, err := ipv4.ParseHeader(buf)
			if err != nil {
				log.Println(err)
				continue
			}
			const ProtocolICMP = 1
			if header.Protocol == ProtocolICMP {
				log.Println("Src:", header.Src, " dst:", header.Dst)
				msg, _ := icmp.ParseMessage(ProtocolICMP, bufs[0][header.Len:])
				log.Println(">> ICMP:", msg.Type)
			}
		}
	}
}
