package main

import (
	"flag"
	"log"
	"net"
	"os"
)

func main() {
	srcPort := flag.String("from", ":4001", "the port data stream come in, usually the swarm master connection")
	destPort := flag.String("to", ":4002", "the port data stream go to, usually the swarm node connection")
	mode := flag.String("mode", "chan", "use as a chan/recv/send")
	flag.Parse()

	switch *mode {
	case "chan":
		log.Printf("srcPort: %s, destPort: %s\n", *srcPort, *destPort)

		log.Println("13")
		channel, err := NewTCPChannel()
		log.Println("14")
		checkErr(err)
		channel.Serve(*srcPort, *destPort)
	case "recv":
		Monitor(*destPort)
	case "send":
		Sender(*srcPort)
	}

	// 旧的示例代码
	// tcpListener, err := net.ListenTCP("tcp", tcpAddr)
	// log.Println("开始监听tcp端口" + tcpAddr.String())
	// HandleTCPAccept(tcpListener)
}

// HandleTCPAccept 处理一个tcpListenser所接收到的conn
func HandleTCPAccept(tcpListenser *net.TCPListener) {
	for {
		if tcpConn, err := tcpListenser.AcceptTCP(); err != nil {
			log.Printf("Accept tcp connectioin err: %s\n", err)
		} else {
			go readTCPConn(tcpConn)
		}

	}
}

// 读取TCP连接中的信息
func readTCPConn(tcpConn *net.TCPConn) {
	// buffer, err := ioutil.ReadAll(tcpConn)
	// if err != nil {
	// 	log.Printf("一个tcp连接[%s]的读取失败，该连接即将关闭\n", tcpConn.RemoteAddr())
	// 	tcpConn.Close()
	// 	return
	// }
	// log.Printf("tcp连接[%s]受到的信息： %s\n", tcpConn.RemoteAddr(), buffer)
	tcpConn.SetReadBuffer(1024)
	buffer := make([]byte, 1024)
	for {
		// n, err := tcpConn.Read(buffer[0:])
		_, err := tcpConn.Read(buffer)

		if err != nil {
			log.Printf("一个tcp连接[%s]的读取失败，该连接即将关闭\n", tcpConn.RemoteAddr())
			tcpConn.Close()
			return
		}
		log.Printf("tcp连接[%s]受到的信息： %s\n", tcpConn.RemoteAddr(), buffer)

		// 为了让对方不会一直保持连接，这里先断开连接
		tcpConn.Close()
	}
}

func checkErr(err error) {
	if err != nil {
		log.Printf("Fatal error : %s", err.Error())
		os.Exit(1)
	}

}
