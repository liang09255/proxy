package main

import (
	"golang.org/x/net/proxy"
	"io"
	"log"
	"net"
	"net/http"
	"time"
)

func main() {
	client := getProxyClient("127.0.0.1", "3000")
	resp, err := client.Get("http://www.baidu.com")
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	log.Println(string(body))
}

func getProxyClient(proxyIP, proxyPort string, auth ...*proxy.Auth) *http.Client {

	proxyurl := proxyIP + ":" + proxyPort
	var author *proxy.Auth = nil
	if auth != nil {
		author = auth[0]
	}
	dialer, err := proxy.SOCKS5("tcp", proxyurl, author,
		&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		},
	)

	if err != nil {
		log.Println("getProxyClient : proxy.SOCKS5, err: ", err)
		return nil
	}

	transport := &http.Transport{
		Proxy:               nil,
		Dial:                dialer.Dial,
		TLSHandshakeTimeout: 10 * time.Second,
	}

	return &http.Client{Transport: transport}
}
