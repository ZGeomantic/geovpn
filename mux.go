package main

import (
	"fmt"
	"log"
	"strings"
)

// Mux 用于在通信的tcp长连接中复用单线路的分发器
type Mux struct {
	router map[string]chan []byte
}

// Dispatch 用于根据传入的头几个字节，来区分该把这条消息分发给route中的那个管道
func (mux *Mux) Dispatch(msg []byte) {
	// ns 取自msg

	ns, body, err := mux.DecodeMsg(msg)
	if err != nil {
		log.Printf("丢弃消息，因为： %s\n", err)
		return
	}
	// 把裁剪掉头部的msg传入相应的router
	channel, ok := mux.router[ns]
	if !ok {
		log.Printf("丢弃消息，因为没有注册该ns: [%s]", ns)
		return
	}
	channel <- body
}

// Add 为某个管道注册ns
func (mux *Mux) Add(ns string, channel chan []byte) {
	mux.router[ns] = channel
}

// Get 根据ns获取相应的管道
func (mux *Mux) Get(ns string) (channel chan []byte) {
	return mux.router[ns]
}

func (mux *Mux) Exist(ns string) bool {
	_, ok := mux.router[ns]
	return ok
}

// EncodeMsg 为消息填上分发区分位ns
func (mux *Mux) EncodeMsg(ns string, raw []byte) []byte {
	return append([]byte(ns), raw...)
}

// DecodeMsg 为把带有ns的消息分解开，分解成ns和body两段
func (mux *Mux) DecodeMsg(raw []byte) (ns string, body []byte, err error) {

	if len(raw) < NS_LENGTH {
		return "", nil, fmt.Errorf("带ns的消息体格式不正确，长度小于NS_LENGTH, 长度：%d", len(body))
	}
	ns = string(raw[:NS_LENGTH])
	body = raw[NS_LENGTH:]
	return ns, body, nil
}

// GetNS 根据对方的地址生成一个位置的NameSpace
func (mux *Mux) GetNS(remoteAddr string) string {
	port := strings.Split(remoteAddr, ":")[1]
	ns := []byte(port)[:NS_LENGTH]
	return string(ns)
}
