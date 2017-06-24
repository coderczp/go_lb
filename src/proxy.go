package main

import (
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

var lock sync.Mutex

func main() {
	args := os.Args
	if args == nil || len(args) < 2 {
		fmt.Println("Usage: -l 0.0.0.0:8000(listen) -s 127.0.0.1:2345,127.0.0.1:2346(backendserver)")
		return
	}

	var addres string
	var server string

	flag.StringVar(&addres, "l", "", "listen")
	flag.StringVar(&server, "s", "", "backendserver")
	flag.Parse()

	serverList := strings.Split(server, ",")
	if len(serverList) <= 0 {
		fmt.Println("backend server is empty")
		return
	}
	fmt.Printf("listen at: %s,backendser: %s\n", addres, server)
	doServer(addres, serverList)
}

/**开始服务*/
func doServer(localAddr string, backendSer []string) {
	lis, err := net.Listen("tcp", localAddr)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer lis.Close()
	for {
		conn, err := lis.Accept()
		if err != nil {
			fmt.Println("accept err:%v\n", err)
			continue
		}
		fmt.Printf("forward :%s<->%s\n",conn.RemoteAddr(), conn.LocalAddr())
		go doProxy(conn, backendSer)
	}
}

/**执行代理逻辑*/
func doProxy(sconn net.Conn, backendSer []string) {
	defer sconn.Close()
	ip := loadBalance(backendSer)
	dconn, err := net.Dial("tcp", ip)
	if err != nil {
		fmt.Printf("dial%v err:%v\n", ip, err)
		return
	}
	ExitChan := make(chan bool, 1)
	go func(sconn net.Conn, dconn net.Conn, Exit chan bool) {
		_, err := io.Copy(dconn, sconn)
		fmt.Printf("write %v data fail:%v\n", ip, err)
		ExitChan <- true
	}(sconn, dconn, ExitChan)
	go func(sconn net.Conn, dconn net.Conn, Exit chan bool) {
		_, err := io.Copy(sconn, dconn)
		fmt.Printf("recv %v data fail:%v\n", ip, err)
		ExitChan <- true
	}(sconn, dconn, ExitChan)
	<-ExitChan
	dconn.Close()
}

/*执行负载均衡*/
func loadBalance(sers []string) string {
	lock.Lock()
	defer lock.Unlock()
	size := len(sers)
	if size == 1 {
		return sers[0]
	}
	now := time.Now().Second()
	return sers[now%size]
}
