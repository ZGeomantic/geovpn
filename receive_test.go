package main

import (
	"fmt"
	"io"
	"net"
	"testing"
	"time"
)

func TestXX(t *testing.T) {
	var mux Mux
	ns := mux.EncodeMsg("127.0.0.1:51000", []byte("abced"))
	t.Logf("result: [%s]\n", ns)
	t.Logf("result: [%v]\n", ns)

	ns1 := string([]byte{53, 49, 0, 0, 0})
	ns2 := string([]byte{53, 49})
	t.Logf("is equal: %v\n", ns1 == ns2)
}
func TestTCP(t *testing.T) {
	addr, err := net.ResolveTCPAddr("tcp", ":5566")
	if err != nil {
		t.Fatalf("解析tcp地址失败： %s\n", err)
	}

	tcpListenser, err := net.ListenTCP("tcp", addr)
	if err != nil {
		t.Fatalf("监听tcp指定地址失败：%s\n", err)
	}
	fmt.Printf("开始监听...\n")
	for {
		conn, err := tcpListenser.AcceptTCP()
		if err != nil {
			t.Fatalf("建立连接失败： %s\n", err)
		}
		fmt.Printf("收到连接...\n")
		go handlerConn(conn, t)
	}
}

// func BenchmarkReadBody(b *testing.B) {
// 	raw := []byte("Replace <SECRET> with the secret that was printed by docker swarm init in the previous step. Replace <MANAGER-IP> with the address of the manager node and <PORT> with the port where the manager listens.")
// 	for i := 0; i < b.N; i++ {
// 		conn := bytes.NewBuffer(raw)
// 		result, err := ReadBodyLocalVar(conn, b)
// 		if err != nil {
// 			b.Fatalf("失败：%s\n", err)
// 		}
// 		if !bytes.Equal(result, raw) {
// 			b.Fatalf("内容不符合： %s\n", result)
// 		}
// 	}
// }

// func BenchmarkReadBodySignal(b *testing.B) {
// 	raw := []byte("Replace <SECRET> with the secret that was printed by docker swarm init in the previous step. Replace <MANAGER-IP> with the address of the manager node and <PORT> with the port where the manager listens.")
// 	for i := 0; i < b.N; i++ {
// 		conn := bytes.NewBuffer(raw)
// 		result, err := ReadBodySignal(conn, b)
// 		if err != nil {
// 			b.Fatalf("失败：%s\n", err)
// 		}

// 		if !bytes.Equal(result, raw) {
// 			b.Fatalf("内容不符合： %s\n", result)
// 		}
// 	}
// }

func handlerConn(conn *net.TCPConn, t *testing.T) {
	// b, err := ioutil.ReadAll(conn)
	// ioutil.ReadAll(conn)
	fmt.Printf("消息来自：%s\n", conn.RemoteAddr())
	// b := make([]byte, 1000)
	// n, err := conn.Read(b)
	b, err := ReadBodyBloom(conn)
	if err != nil {
		t.Fatalf("处理请求失败：%s\n", err)
	}

	// fmt.Printf("读取到消息体：\n%s， 长度： %d\n", b, n)
	fmt.Printf("读取到消息体：\n%s\n", b)
	fmt.Printf("读取到消息体：\n%v\n", b[:])

	// fmt.Printf("读取到消息体：\n%v\n", b[:n])

	conn.Close()
}

// 这是错误的方法
func ReadBodyBloom(conn io.Reader) (content []byte, err error) {
	buffsize := 10

	b := make([]byte, buffsize)

	readChan := make(chan []byte)

	go func() {
		for {
			// 并不是readChan里放入了值之后，就立刻转让控制权到执行<-readChan的操作，本线程会执行到下一次阻塞的地方，
			// 具体来说，当readChan里的值被读取走之后，下面的select语句阻塞住，程序调整到这里执行，把b上一次的[:n]放入chan中，
			// 然后又读取了新的内容，准备放入chan里，但是由于这两个slice指向同一个地址，所以这两个次让对方收到的内容重复的，
			n, err := conn.Read(b[0:])
			fmt.Printf("read length: %d - [%s]\n", n, b[0:])

			if err != nil {
				close(readChan)
				return
			}
			readChan <- b[:n]

			if n < buffsize {
				close(readChan)
				return
			}
		}

	}()

	for {
		select {
		case c, ok := <-readChan:
			if ok {
				fmt.Printf("receive: %s\n", c)
				content = append(content, c...)
			} else {
				fmt.Println("read closed chan")
				goto OUTSIDE
			}
		case <-time.After(time.Millisecond * 100):
			fmt.Println("break!")
			goto OUTSIDE
		}
	}

OUTSIDE:
	fmt.Printf("length: %d\n", len(content))

	return content, nil
}
func TestStrange(t *testing.T) {
	signal := make(chan int)

	go func() {
		for {
			select {
			case v := <-signal:
				fmt.Printf("***receive: %d\n", v)
			}
		}
	}()

	time.Sleep(time.Second)

	// go func() {
	for i := 1; i <= 6; i++ {
		fmt.Printf("put %d\n", i)
		signal <- i
		fmt.Printf("end put %d\n", i)

	}
	// }()
	closed := make(chan int)
	<-closed
}

func TestNew(t *testing.T) {
	closed := make(chan int)
	a := make(chan int)
	go func() {
		fmt.Println("100")
		closed <- 1
		fmt.Println("101")
		closed <- 1
		fmt.Println("102")

	}()

	fmt.Println("1")
	<-closed
	// <-closed

	fmt.Println("2")

	<-closed
	fmt.Println("3")
	<-a

}

// 通过循环声明新的局部变量，来解决多线程中值被覆盖的问题
func ReadBodyLocalVar(conn io.Reader, t *testing.T) (content []byte, err error) {
	buffsize := 10
	b := make([]byte, buffsize)

	readChan := make(chan []byte)
	go func() {
		for {
			// 这个b不能放在外面声明为全局变量，需要每一次循环里重新创建，原因下面的注释
			// b := make([]byte, buffsize)

			n, err := conn.Read(b[0:])
			fmt.Printf("read length: %d - [%s]\n", n, b[0:])

			if err != nil {
				close(readChan)
				return
			}
			// 这里放入管道的slice是一个指针类型，如果不每次创建一个新的局部变量，
			// 在下一次执行到阻塞的代码之前，也就是这里的管道阻塞的地方，会覆盖掉之前已经放进管道里的值
			// 会出现的现象之一就是整个content被读取的第一部分长度为buffsize的内容丢失了，原因就是被第二部分覆盖了
			// 具体的步骤是，当第一次执行到这里的时候，第一部分的内容raw[0:buffsize]放进去了，然后等待在这里一直到select语句读出来，阻塞在第二次select循环处。
			// 这时候，readChan阻塞打开，循环继续，可以继续读取第二部分内容raw[buffsize:2*buffsize]到b中，
			readChan <- b[:n]
			fmt.Printf("read length xx:  %d - [%s]\n", n, b[:n])

			if n < buffsize {
				close(readChan)
				return
			}
		}

	}()
	fmt.Println("关键点")
	for {
		select {
		case c, ok := <-readChan:
			if ok {
				fmt.Printf("receive: %s\n", c)
				content = append(content, c...)
			} else {
				t.Log("read closed chan")
				goto OUTSIDE
			}
		case <-time.After(time.Millisecond * 100):
			t.Log("break!")
			goto OUTSIDE
		}
	}

OUTSIDE:
	t.Logf("length: %d\n", len(content))

	return content, nil
}

// 通过信号灯，来解决多线程中值被覆盖的问题
func ReadBodySignal(conn io.Reader, t *testing.B) (content []byte, err error) {
	buffsize := 10
	// 这里设置长度为1，是为了下面的放入启动信号的逻辑不被阻塞
	// 这个signal是为了保证b为全局变量，还不会因为复用问题后面覆盖前面的
	signal := make(chan int, 1)
	signal <- 0
	b := make([]byte, buffsize)

	readChan := make(chan []byte)
	go func() {
		for {
			// 这个b不能放在外面声明为全局变量，需要每一次循环里重新创建，原因下面的注释
			<-signal
			n, err := conn.Read(b[0:])
			t.Logf("read length: %d - [%s]\n", n, b[0:])

			if err != nil {
				close(readChan)
				return
			}
			// 这里放入管道的slice是一个指针类型，如果不每次创建一个新的局部变量，
			// 在下一次执行到阻塞的代码之前，也就是这里的管道阻塞的地方，会覆盖掉之前已经放进管道里的值
			// 会出现的现象之一就是整个content被读取的第一部分长度为buffsize的内容丢失了，原因就是被第二部分覆盖了
			readChan <- b[:n]

			if n < buffsize {
				close(readChan)
				return
			}
		}

	}()

	for {
		select {
		case c, ok := <-readChan:
			if ok {
				t.Logf("receive: %s\n", c)
				content = append(content, c...)
				signal <- 1
			} else {
				t.Log("read closed chan")
				goto OUTSIDE
			}
		case <-time.After(time.Millisecond * 100):
			t.Log("break!")
			goto OUTSIDE
		}
	}

OUTSIDE:
	t.Logf("length: %d\n", len(content))

	return content, nil
}

func ReadBody(conn *net.TCPConn) (content []byte, err error) {
	buffsize := 10
	b := make([]byte, buffsize)

	for {

		n, err := conn.Read(b)
		fmt.Printf("hawa, %d\n", n)

		if err != nil {
			return nil, err
		}

		content = append(content, b[:n]...)

		if n < buffsize {
			break
		}

	}

	fmt.Printf("length: %d\n", len(content))
	return content, nil
}
