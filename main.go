package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"runtime"
)

var runApp = flag.String("exec", "", "execute app.\t\tex: bc")

func main() {
	proto := flag.String("p", "tcp", "protocol.\t\ttcp or udp")
	listenOn := flag.String("l", "", "listen on port.\t\tex: -l :3000")
	connectTo := flag.String("c", "", "connect to address.\tex: -c 127.0.0.1:3000")
	shellMode := flag.Bool("s", false, "shell mode")
	flag.Parse()

	if *listenOn == *connectTo && *listenOn == "" {
		flag.Usage()
		os.Exit(1)
	}

	if *shellMode {
		*runApp = "/bin/bash"
		if runtime.GOOS == "windows" {
			*runApp = "cmd.exe"
		}
	}
	protoList := map[string]bool{"tcp": true, "udp": true}
	if !protoList[*proto] {
		_, _ = fmt.Fprintln(os.Stderr, "invalid protocol:", *proto)
		os.Exit(1)
	}

	if *listenOn != "" {
		_ = runAsServer(*proto, *listenOn)
	} else {
		_ = runAsClient(*proto, *connectTo)
	}

}

func handleConnection(conn net.Conn) {
	AddClient(conn)
	defer func() {
		RemoveClient(conn)
		_ = conn.Close()
	}()
	fmt.Println(conn.RemoteAddr(), "Connected!")
	HandleStdin(conn)
	Dispatch(conn, WriterIO)
	fmt.Println(conn.RemoteAddr(), "Disconnected!")
}

func runAsServer(proto string, addr string) error {
	fmt.Printf("Running server on %s://%s\r\n", proto, addr)
	l, err := net.Listen(proto, addr)
	if err != nil {
		return err
	}
	defer func() { _ = l.Close() }()
	fmt.Println("Waiting for connection...")
	for {
		conn, err := l.Accept()
		if err != nil {
			log.Fatal(err)
		}
		go handleConnection(conn)
	}
	return nil
}

func runAsClient(proto string, addr string) error {
	fmt.Printf("Connecting to %s://%s\r\n", proto, addr)
	conn, err := net.Dial(proto, addr)
	if err != nil {
		log.Fatal(err)
	}
	handleConnection(conn)
	return nil
}
