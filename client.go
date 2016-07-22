package main

import (
	"log"
	"net"
)

// Client 作为客户端模式，把收到的数据，转发给指定的http端都
func Client(remoteAddr, httpAddr string) {
	conn, err := net.Dial("tcp", remoteAddr)

	if err != nil {
		log.Printf("连接远程tcp端口失败： %s", err)
		return
	}

	data := make(chan []byte)
	go connectHTTP(httpAddr, data)

	buffer := make([]byte, 1024)
	for {
		_, err = conn.Read(buffer)
		if err != nil {
			log.Printf("读取数据出错： %s", err)
			continue
		}
		log.Printf("读取数据： %s", buffer)
		data <- buffer
		log.Printf("读取的数据已经转发给http server")
		conn.Write(<-data)
		log.Printf("收到http server响应")

	}
}

func connectHTTP(httpAddr string, data chan []byte) error {
	conn, err := net.Dial("tcp", httpAddr)

	if err != nil {
		log.Printf("http请求，以tcp方式连接端口失败： %s", err)
		return err
	}

	for {
		content := <-data
		log.Printf("准备发送http请求")
		_, err = conn.Write(content)
		if err != nil {
			log.Printf("发送http数据失败： %s", err)
			return err
		}

		log.Printf("准备接受http响应")
		resp := make([]byte, 1024)
		_, err = conn.Read(resp)
		data <- resp
		if err != nil {
			log.Printf("接收http数据失败： %s", err)
			return err
		}
	}

}
