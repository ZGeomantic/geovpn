package main

import (
	"log"
	"net"
)

// ConnAgent 代表着线上服务器把http请求封装成tcp的请求
type ConnAgent struct {
	// bridgeDownStream 表示连接线下的client的TCP长连接中的发送操作的管道
	bridgeDownStream chan []byte
	bridgeUpStream   chan []byte
	bridgeConn       *net.TCPConn
	mux              *Mux
}

// 处理server和client之间的长连接
func (ag *ConnAgent) bridgePump() {

	go readerPump(ag.bridgeConn, ag.bridgeUpStream)
	// go writerPump(agent.bridgeConn, agent.bridgeDownStream)
	log.Println("Agent开始监听连接")
	for {
		select {
		case upstream, ok := <-ag.bridgeUpStream:
			if !ok {
				log.Println("bridge的bridgeUpStream关闭，Agent也即将暂停")
				return
			}
			log.Printf("bridge收到上传数据：%s\n", upstream)
			// 把bridge返回的响应分发到对应的conn的write chan中，然后conn会自动把属于自己write chan的内容写回conn
			ag.mux.Dispatch(upstream)

		case downstream, ok := <-ag.bridgeDownStream:
			if !ok {
				log.Println("bridge的bridgeDownStream关闭，Agent也即将暂停")
				return
			}

			log.Printf("bridge下发数据： %s\n", downstream)
			ag.bridgeConn.Write(downstream)
		}
	}
}

// 建立线上和线下的长连接
func (ag *ConnAgent) establishBridge(bridgeAddr string) error {
	tcpListener, err := tcpListenserFactory(bridgeAddr)
	if err != nil {
		log.Printf("bridge创建监听器失败： %s\n", err)
	}

	log.Println("Agent等待bridge建立连接...")
	for {
		conn, err := tcpListener.AcceptTCP()
		if err != nil {
			log.Printf("bridge建立连接失败： %s\n", err)
			continue
		}
		ag.bridgeConn = conn
		ag.bridgeUpStream = make(chan []byte)
		ag.bridgeDownStream = make(chan []byte)
		go ag.bridgePump()
	}
}

// ServeTCP 实现了TCPHandler接口，
func (ag *ConnAgent) ServeTCP(conn *net.TCPConn) {
	ns := ag.mux.GetNS(conn.RemoteAddr().String())
	readChan := make(chan []byte)
	writeChan := make(chan []byte)

	go readerPump(conn, readChan)

	// 这里把writechan注册到mux中，在bridge收到upstream的消息的时候，会把消息写入这个writechan中，(见bridgePump的agent.mux.Dispatch(upstream))
	// 传递给writechan的内容，都会原封不动的发送到conn中。
	// 所以写操作没有在这里显式的编码，而是通过writerPump的数据流动自动完成
	ag.mux.Add(ns, writeChan)
	go writerPump(conn, writeChan)

	for {
		select {
		case req, ok := <-readChan:
			if !ok {
				log.Printf("[%s]的TCP的readChan已经关闭", ns)
				return
			}
			// TODO：接收到的数据，加一个ns后转发给bridge
			ag.bridgeDownStream <- ag.mux.EncodeMsg(ns, req)
		}
	}
}

// StartConnAgent 用于构造ConnAgent
func StartConnAgent(httpPort, bridgeAddr string) {
	agent := &ConnAgent{
		bridgeDownStream: make(chan []byte),
		bridgeUpStream:   make(chan []byte),
		mux: &Mux{
			router: make(map[string]chan []byte),
		},
	}

	// 打通server与client的TCP长连接工作
	go agent.establishBridge(bridgeAddr)

	startTCPHandler(httpPort, agent)

}
