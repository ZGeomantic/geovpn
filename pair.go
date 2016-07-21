package main

import (
	"io"
	"log"
	"net"
	"os"
	"sync"
)

// TCPChannel 代表一对tcp连接，数据从一个socket流向另一个
type TCPChannel struct {
	srcConn  *net.TCPConn
	destConn *net.TCPConn
	wg       sync.WaitGroup
}

// SetSrc 设置了源conn，这个连接应该是一个Server端口，
func (tc *TCPChannel) SetSrc(addr *net.TCPAddr) error {
	conn, err := tc.acceptConn(addr)

	if err != nil {
		log.Println("与源端口建立连接失败")
		return err
	}
	tc.srcConn = conn
	log.Println("src端口已经准备就绪")
	tc.wg.Done()
	return nil
}

// SetDest 设置了目的conn，此连接在启动后应该已由client端主动连接上，然后
func (tc *TCPChannel) SetDest(addr *net.TCPAddr) error {

	conn, err := tc.acceptConn(addr)
	if err != nil {
		log.Println("与目的端口建立连接失败")
		return err
	}
	tc.destConn = conn
	log.Println("dest端口已经准备就绪")
	tc.wg.Done()
	return nil
}

// Serve 表示两个tcp连接已经建立，可以开始传输数据了
func (tc *TCPChannel) Serve() {
	log.Println("准备传输数据")
	tc.wg.Add(2)
	tc.wg.Wait()
	log.Println("开始传输数据")
	num, err := io.Copy(tc.destConn, tc.srcConn)
	log.Printf("总共传输%d字节，关闭：%s\n", num, err)
	tc.srcConn.Close()
	tc.destConn.Close()
}

// 从一个监听地址上获取连接对象
func (tc *TCPChannel) acceptConn(addr *net.TCPAddr) (*net.TCPConn, error) {
	log.Println("31")

	tcpListenser, err := net.ListenTCP("tcp", addr)
	if err != nil {
		log.Printf("监听端口%s失败,错误信息：%s\n", addr, err)
		return nil, err
	}
	log.Println("32")

	//这里是个阻塞的函数
	tcpConn, err := tcpListenser.AcceptTCP()
	if err != nil {
		log.Printf("Accept tcp connectioin err: %s\n", err)
		return nil, err
	}
	log.Println("33")

	return tcpConn, nil
}

// NewTCPChannel 通过一个listenser创建一个TCPChannel
func NewTCPChannel(srcAddr, destAddr *net.TCPAddr) (*TCPChannel, error) {
	log.Println("21")

	channel := new(TCPChannel)
	log.Println("22")

	go channel.SetSrc(srcAddr)
	log.Println("23")

	go channel.SetDest(destAddr)
	log.Println("24")

	return channel, nil
}

func startChannel(srcAddr, destAddr *net.TCPAddr) {
	channel, err := NewTCPChannel(srcAddr, destAddr)
	if err != nil {
		os.Exit(1)
	}
	channel.Serve()
}

// Monitor 用于仅作为接收端，显示数据，不再转发
func Monitor(addr string) {
	conn, err := net.Dial("tcp", addr)

	if err != nil {
		log.Printf("连接远程tcp端口失败： %s", err)
	}

	buffer := make([]byte, 1024)
	for {
		_, err = conn.Read(buffer)
		if err != nil {
			log.Printf("读取数据出错： %s", err)
			os.Exit(1)
		}
		log.Printf("读取数据： %s", buffer)

	}

}

// Sender 用于不断发送数据
func Sender(addr string) {
	conn, err := net.Dial("tcp", addr)

	if err != nil {
		log.Printf("连接远程tcp端口失败： %s", err)
	}
	data := []byte("hello world")
	for {
		_, err = conn.Write(data)
		log.Printf("写数据： %s", data)
	}

}
