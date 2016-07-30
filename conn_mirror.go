package main

import (
	"log"
	"net"
)

// ConnMirror 与ConnAgent相对应，是它在线下的镜像
type ConnMirror struct {
	// bridgeDownStream 表示连接线下的client的TCP长连接中的发送操作的管道
	bridgeDownStream chan []byte
	bridgeUpStream   chan []byte
	bridgeConn       net.Conn
	mux              *Mux
	localHTTPAddr    string
}

// StartConnMirror 启动函数
func StartConnMirror(offlineHttpAddr, bridgeAddr string) {
	connMirror := ConnMirror{
		bridgeDownStream: make(chan []byte),
		bridgeUpStream:   make(chan []byte),
		mux: &Mux{
			router: make(map[string]chan []byte),
		},
		localHTTPAddr: offlineHttpAddr,
	}

	conn, err := net.Dial("tcp", bridgeAddr)
	if err != nil {
		log.Printf("client无法连接上agent： %s\n", err)
		return
	}
	connMirror.bridgeConn = conn
	connMirror.bridgePump()
}

func (cm *ConnMirror) bridgePump() {
	go readerSegmentPump(cm.bridgeConn, cm.bridgeDownStream)
	go writerSegmentPump(cm.bridgeConn, cm.bridgeUpStream)

	for {
		select {
		case msg, ok := <-cm.bridgeDownStream:
			if !ok {
				log.Println("bridgeDownStream关闭，ConnMirror也即将暂停")
				return
			}
			log.Printf("client收到msg: [%s]\n", msg)
			ns, _, err := cm.mux.DecodeMsg(msg)

			if err == nil && !cm.mux.Exist(ns) {
				reqChan := make(chan []byte)
				// respChan := make(chan []byte)

				worker := MirrorWorker{
					ns:             ns,
					bridgeReqChan:  reqChan,
					httpRespChan:   make(chan []byte),
					bridgeUpStream: cm.bridgeUpStream,
					httpAddr:       cm.localHTTPAddr,
					mux:            cm.mux,
				}
				cm.mux.Add(ns, reqChan)
				go worker.Loop()
			}

			cm.mux.Dispatch(msg)
		}
	}
}

// MirrorWorker 表示运行在ConnMirror中的多个协程中的http连接任务
type MirrorWorker struct {
	ns string
	// reqChan 用于接收来自bridge的信息，通过mux分发
	bridgeReqChan chan []byte
	// httpRespChan 用于client接受本地http服务器的响应
	httpRespChan   chan []byte
	bridgeUpStream chan []byte
	httpAddr       string
	mux            *Mux
}

// Loop 在一个http请求没有断开的期间内，这个函数一直保持着agent和mirror之间的通信
func (mw *MirrorWorker) Loop() {
	conn, err := net.Dial("tcp", mw.httpAddr)

	if err != nil {
		log.Printf("连接http服务器失败： %s\n", err)
		return
	}
	go readerPump(conn, mw.httpRespChan)

	for {
		select {
		case msg := <-mw.bridgeReqChan:
			// 这里相当于转发http请求
			conn.Write(msg)

		case resp, ok := <-mw.httpRespChan:
			if !ok {
				log.Printf("worker[%s]已关闭", mw.ns)
				return
			}
			log.Printf("收到client端返回的响应：%s\n", resp)
			mw.bridgeUpStream <- mw.mux.EncodeMsg(mw.ns, resp)
		}

	}
}
