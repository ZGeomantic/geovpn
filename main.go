package main

import (
	"flag"
	"log"
)

const (
	READ_BUFFER_SIZE = 1024
)

func main() {
	log.SetFlags(log.Lshortfile | log.Ltime)
	// 总参数
	mode := flag.String("mode", "chan", "use as a chan/recv/send")
	bridgePort := flag.String("bridge_port", ":4002", "the port that long TCP connection listensing on")

	// 服务端参数
	httpPort := flag.String("p", ":4001", "the port data stream come in, usually the swarm master connection")

	// 客户端参数
	offlineHTTPAddr := flag.String("offline_http", ":4000", "the client use to connect offline http server")
	// vpnAddr := flag.String("remote_bridge_address", ":4002", "")// 可以和bridgePort参数合并

	flag.Parse()
	log.Printf("httpPort: %s, bridgePort: %s\n", *httpPort, *bridgePort)

	switch *mode {
	case "chan":
		StartConnAgent(*httpPort, *bridgePort)
	case "client":
		StartConnMirror(*offlineHTTPAddr, *bridgePort)
	}

}
