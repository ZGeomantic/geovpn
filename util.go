package main

import (
	"io"
	"log"
	"net"
)

// TCPHandler 定义了处理TCP连接需要实现的接口
type TCPHandler interface {
	ServeTCP(conn *net.TCPConn)
}

func tcpListenserFactory(tcpAddr string) (*net.TCPListener, error) {
	addr, err := net.ResolveTCPAddr("tcp", tcpAddr)
	if err != nil {
		log.Printf("解析tcp地址[%s]失败： %s\n", tcpAddr, err)
		return nil, err
	}

	tcpListenser, err := net.ListenTCP("tcp", addr)
	if err != nil {
		log.Printf("监听端口%s失败,错误信息：%s\n", addr, err)
		return nil, err
	}
	return tcpListenser, nil
}

func startTCPHandler(tcpAddr string, handler TCPHandler) error {
	log.Println("开始监听tcp")
	tcpListenser, err := tcpListenserFactory(tcpAddr)
	if err != nil {
		return err
	}
	for {
		conn, err := tcpListenser.AcceptTCP()
		if err != nil {
			log.Printf("建立TCP连接错误: %s\n", err)
			continue
		}

		go handler.ServeTCP(conn)
	}
}

func readStream(conn net.Conn, readChan chan []byte, readlength int) {
	for {
		buffer := make([]byte, readlength)
		n, err := conn.Read(buffer)

		if err != nil {
			if err == io.EOF {
				log.Printf("读取连接结束\n")
			} else {
				log.Printf("读取连接失败： %s\n", err)
			}
			log.Printf("连接[%s]即将关闭\n", conn.RemoteAddr())
			conn.Close()
			close(readChan)
			return
		}
		readChan <- buffer[:n]
	}
}

// readerPump 把tcp连接中的内容全部读取到管道readChan中
func readerPump(conn net.Conn, readChan chan []byte) {
	readStream(conn, readChan, READ_BUFFER_SIZE)
}

// readerSegmentPump 用于读取带ns头部的内容，用于在收发通过内部bridge通信的tcp数据包
func readerSegmentPump(conn net.Conn, readChan chan []byte) {
	readStream(conn, readChan, READ_BUFFER_SIZE+NS_LENGTH)
}

// 把writeChan中的内容作为tcp连接的返回内容
func writerPump(conn net.Conn, writeChan chan []byte) {
	for {
		bs := <-writeChan

		if _, err := conn.Write(bs); err != nil {
			log.Printf("连接写数据失败： %s\n", err)
			log.Printf("此连接即将关闭\n")
			conn.Close()
			close(writeChan)
			return
		}

	}
}

func writerSegmentPump(conn net.Conn, writeChan chan []byte) {
	for {
		bs := <-writeChan
		for i := 0; i < len(bs); i = i + WRITE_BUFFER_SIZE {
			endPos := i + WRITE_BUFFER_SIZE
			if i+WRITE_BUFFER_SIZE >= len(bs) {
				endPos = len(bs)
			}
			if _, err := conn.Write(bs[i:endPos]); err != nil {
				log.Printf("连接写数据失败： %s\n", err)
				log.Printf("连接[%s]即将关闭\n", conn.RemoteAddr())
				conn.Close()
				close(writeChan)
				return
			}
		}

	}
}
