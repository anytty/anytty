// Command remote-bridge forwards a loopback TCP port to a unix socket byte
// for byte. It is the portable stand-in for
//
//	ssh -N -L 127.0.0.1:17777:/run/user/1000/anytty/daemon.sock host
//
// when ssh is unavailable (for example when the "remote" daemon already runs
// on this machine). Both forms terminate at the daemon framed transport, so a
// tui2 endpoint can use connect_mode "tcp" with address 127.0.0.1:17777.
//
// Usage: remote-bridge -listen 127.0.0.1:17777 -unix /path/to/anytty.sock
package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	listen := flag.String("listen", "127.0.0.1:17777", "TCP listen address (loopback recommended)")
	unixPath := flag.String("unix", "", "unix socket path of the terminal pool transport")
	flag.Parse()
	if *unixPath == "" {
		log.Fatal("remote-bridge: -unix is required")
	}
	ln, err := net.Listen("tcp", *listen)
	if err != nil {
		log.Fatalf("remote-bridge: listen %s: %v", *listen, err)
	}
	fmt.Fprintf(os.Stderr, "remote-bridge: listening %s -> unix %s\n", ln.Addr(), *unixPath)

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-signals
		_ = ln.Close()
	}()

	for {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		go forward(conn, *unixPath)
	}
}

func forward(conn net.Conn, unixPath string) {
	defer conn.Close()
	upstream, err := net.DialTimeout("unix", unixPath, 5*time.Second)
	if err != nil {
		return
	}
	defer upstream.Close()
	done := make(chan struct{})
	go func() {
		_, _ = io.Copy(upstream, conn)
		if closer, ok := upstream.(interface{ CloseWrite() error }); ok {
			_ = closer.CloseWrite()
		}
		close(done)
	}()
	_, _ = io.Copy(conn, upstream)
	<-done
}
