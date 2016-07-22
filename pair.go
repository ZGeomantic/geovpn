package main

import (
	"log"
	"net"
	"os"
	"time"
)

// TCPChannel 代表一对tcp连接，数据从一个socket流向另一个
type TCPChannel struct {
	onlineReq chan []byte
	// onlineResp chan []byte
	// offlineReq  chan []byte
	offlineResp chan []byte

	closeEvent chan int
}

// ServeOnline 设置了源conn，这个连接应该是一个Server端口，
func (tc *TCPChannel) ServeOnline(addr string) error {
	listenser, err := tc.getListener(addr)
	if err != nil {
		log.Println("与线上端口建立连接失败")
		return err
	}
	go tc.acceptReq(listenser)
	log.Println("线上端口开始工作")
	return nil
}

// ServeOffline 设置了目的conn，此连接在启动后应该已由client端主动连接上，然后
func (tc *TCPChannel) ServeOffline(addr string) error {
	listenser, err := tc.getListener(addr)
	if err != nil {
		log.Println("与线下端口建立连接失败")
		return err
	}
	go tc.acceptResp(listenser)
	log.Println("线下端口开始工作")
	return nil
}

// Serve 表示两个tcp连接已经建立，可以开始传输数据了
func (tc *TCPChannel) Serve(srcAddr, destAddr string) {
	log.Println("启动channel端")
	tc.ServeOnline(srcAddr)

	tc.ServeOffline(destAddr)
	log.Println("启动完毕")
	<-tc.closeEvent
}

// 解析出tcp的监听地址，并构造监听器的辅助函数
func (tc *TCPChannel) getListener(tcpAddr string) (*net.TCPListener, error) {
	log.Println("31")

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

// 接受公网传来的请求数据，传递给线下
func (tc *TCPChannel) acceptReq(tcpListenser *net.TCPListener) {
	//这里是个阻塞的函数
	for {
		tcpConn, err := tcpListenser.AcceptTCP()
		if err != nil {
			log.Printf("Accept tcp online connectioin err: %s\n", err)
			continue
		}
		buffer := make(chan []byte)
		go readConn(tcpConn, buffer, "线上")
		log.Println("线上通道建立成功，准备传输数据")

		// go func() {
		// 	for {
		// 		log.Printf("临时处理一下tc.onlineReq，否则会阻塞。注意，如果不是仅仅打通onlineReq，一定要记得把这个注释掉，否则争抢了chan，正常逻辑会阻塞： %s", <-tc.onlineReq)
		// 	}
		// }()

		for {
			log.Println("--------------")
			select {
			case req, ok := <-buffer:
				if !ok {
					goto CLOSE
				}
				log.Printf("【线上端口】收到请求数据%s\n", req)
				tc.onlineReq <- req
				log.Printf("转发到线下\n")

				// 这种写法虽然看起来吊，但是太危险了，因为两个管道中不论哪一个阻塞了，就卡主不动了
				// 这样写就是会阻塞，但是改成下面这样就不会阻塞。即使明明逻辑并没有执行到这里，也会阻塞，还不明白具体的原因
				// case tc.onlineResp <- <-tc.offlineResp:
			case resp := <-tc.offlineResp:
				log.Printf("【线上端口】收到【线下端口】的响应内容： %s\n", resp)
				tcpConn.Write(resp)

			}
			log.Println("============")

		}
	CLOSE:
		log.Println("连接断开，等待重新建立连接...")
	}

}

// 接受线下传来的响应数据，传递给线上
func (tc *TCPChannel) acceptResp(tcpListenser *net.TCPListener) {

	for {
		tcpConn, err := tcpListenser.AcceptTCP()
		if err != nil {
			log.Printf("Accept tcp offline connectioin err: %s\n", err)
			continue
		}

		buffer := make(chan []byte)
		go readConn(tcpConn, buffer, "线下")
		log.Println("线下通道建立成功，准备传输数据")

		for {
			select {
			case req := <-tc.onlineReq:
				// tc.offlineReq <- req
				log.Printf("【线下端口】收到req： %s\n", req)
				tcpConn.Write(req)
				// log.Printf("收到来自线上的转发数据，将要往线下写数据: %s", <-tc.offlineReq)

			case resp, ok := <-buffer:
				if !ok {
					goto CLOSE
				}
				tc.offlineResp <- resp
				log.Printf("【线下端口】收到【client】返回的数据：%s", resp)
			}
		}
	CLOSE:
		log.Println("连接断开，等待重新建立连接...")
	}
}

func readConn(conn *net.TCPConn, buffer chan []byte, name string) {
	readBuffer := make([]byte, 1024)

	for {
		n, err := conn.Read(readBuffer)
		if err != nil {
			log.Printf("读取TCP Connection失败： %s", err)
			buffer <- readBuffer
			conn.Close()
			close(buffer)
			return
		}
		log.Printf("[%s]收到数据： %s\n", name, readBuffer)
		_ = n
		buffer <- readBuffer
	}
}

// NewTCPChannel 通过一个listenser创建一个TCPChannel
func NewTCPChannel() (*TCPChannel, error) {

	channel := &TCPChannel{
		// offlineReq:  make(chan []byte),
		offlineResp: make(chan []byte),
		onlineReq:   make(chan []byte),
		// onlineResp:  make(chan []byte),
		closeEvent: make(chan int),
	}
	return channel, nil
}

// Monitor 用于仅作为接收端，显示数据，不再转发
func Monitor(addr string) {
	conn, err := net.Dial("tcp", addr)

	if err != nil {
		log.Printf("连接远程tcp端口失败： %s", err)
		return
	}

	buffer := make([]byte, 1024)
	for {
		_, err = conn.Read(buffer)
		if err != nil {
			log.Printf("读取数据出错： %s", err)
			os.Exit(1)
		}
		log.Printf("读取数据： %s", buffer)
		conn.Write([]byte("哈哈，我收到了"))
	}

}

// Sender 用于不断发送数据
func Sender(addr string) {
	conn, err := net.Dial("tcp", addr)

	if err != nil {
		log.Printf("连接远程tcp端口失败： %s", err)
		return
	}
	data := []byte("hello world")
	for {
		<-time.NewTicker(time.Second).C
		_, err = conn.Write(data)
		log.Printf("写数据： %s", data)
		if err != nil {
			log.Printf("发送数据失败： %s", err)
			return
		}

		resp := make([]byte, 1024)
		conn.Read(resp)
		log.Printf("收到响应： %s\n", resp)
	}

}
