package main

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"github.com/rs/zerolog"
	"io"
	"net"
	"os"
	"runtime"
	"sync"
	"time"
)

const socks5Version = 5
const cmdBind = 1
const (
	atypIPV4 = 1
	atypHOST = 3
	atypIPV6 = 4
)

var logger zerolog.Logger

func init() {
	logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.StampMilli}).With().Timestamp().Caller().Logger()
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnixMs
}

func main() {
	port := "7070"
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		logger.Println("Error listening:", err.Error())
		logger.Fatal()
		return
	}
	defer listener.Close()
	logger.Info().Str("端口", port).Msg("代理开启")

	for {
		conn, err := listener.Accept()
		if err != nil {
			logger.Error().Err(err).Msg("端口监听失败")
			return
		}
		logger.Info().Msg("接收到一个连接")
		go handleConnection(conn)
		logger.Info().Int("数量", runtime.NumGoroutine()).Msg("goroutine")
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)
	// +----+----------+----------+
	// |VER | NMETHODS | METHODS  |
	// +----+----------+----------+
	// | 1  |    1     | 1 to 255 |
	// +----+----------+----------+
	version, err := reader.ReadByte()
	if err != nil {
		logger.Printf("read version failed:%v", err)
		return
	}
	if version != socks5Version {
		logger.Printf("not supported version:%v", version)
		return
	}
	methodSize, err := reader.ReadByte()
	if err != nil {
		logger.Printf("read methodSize failed%v", err)
		return
	}
	method := make([]byte, methodSize)
	_, err = io.ReadFull(reader, method)
	if err != nil {
		logger.Printf("read method failed%v", err)
		return
	}
	logger.Info().Int("Version", int(version)).Bytes("Method", method).Msg("连接信息")
	// +----+--------+
	// |VER | METHOD |
	// +----+--------+
	// | 1  |   1    |
	// +----+--------+
	_, err = conn.Write([]byte{socks5Version, 0x00})
	if err != nil {
		logger.Err(err).Msg("响应失败")
		return
	}
	logger.Info().Msg("协商完成")

	buff := make([]byte, 4)
	_, err = io.ReadFull(reader, buff)
	if err != nil {
		logger.Err(err).Msg("头部信息读取失败")
		return
	}
	ver, cmd, atyp := buff[0], buff[1], buff[3]
	if ver != socks5Version {
		logger.Error().Msg("暂不支持的协议类型")
		return
	}
	if cmd != cmdBind {
		logger.Error().Msg("不支持的cmd类型")
		return
	}
	addr := ""
	switch atyp {
	case atypIPV4:
		_, err = io.ReadFull(reader, buff)
		if err != nil {
			logger.Err(err).Msg("读取ipv4地址失败")
			return
		}
		addr = fmt.Sprintf("%d.%d.%d.%d", buff[0], buff[1], buff[2], buff[3])
	case atypHOST:
		hostSize, err := reader.ReadByte()
		if err != nil {
			logger.Err(err).Msg("读取域名长度失败")
			return
		}
		host := make([]byte, hostSize)
		_, err = io.ReadFull(reader, host)
		if err != nil {
			logger.Err(err).Msg("读取域名失败")
			return
		}
		addr = string(host)
	case atypIPV6:
		logger.Error().Int("atyp", atypIPV6).Msg("暂不支持的地址类型")
		return
	default:
		logger.Error().Int("atyp", int(atyp)).Msg("暂不支持的地址类型")
		return
	}
	_, err = io.ReadFull(reader, buff[:2])
	if err != nil {
		logger.Err(err).Msg("端口读取失败")
		return
	}
	port := binary.BigEndian.Uint16(buff[:2])
	logger.Info().Str("Addr", addr).Uint16("port", port).Msg("目标连接地址")

	_, err = conn.Write([]byte{0x05, 0x00, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
	if err != nil {
		logger.Err(err).Msg("响应失败")
		return
	}

	dest, err := net.Dial("tcp", fmt.Sprintf("%v:%v", addr, port))
	if err != nil {
		logger.Err(err).Msg("建立与目标地址的tcp连接失败")
		return
	}

	wg := sync.WaitGroup{}
	wg.Add(2)

	go func() {
		defer wg.Done()
		io.Copy(dest, reader)
		dest.Close()
	}()
	go func() {
		defer wg.Done()
		io.Copy(conn, dest)
		conn.Close()
	}()

	wg.Wait()
}
