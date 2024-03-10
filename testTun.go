package main

import (
	"golang.zx2c4.com/wireguard/tun"
	"golang.zx2c4.com/wireguard/windows/tunnel/winipcfg"
	"net/netip"
	"time"
)

const DefaultMTU2 = 1420

func main() {
	tun := initTun2("myTun2", "30.0.0.1/24")
	defer tun.Close()
	for {
		time.Sleep(10 * time.Second)
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
