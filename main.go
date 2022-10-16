package main

import (
	"context"
	"flag"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	pool "github.com/MeteorsLiu/go-tcpConnectionPool/v2"
)

func run(it chan struct{}, is context.Context, l net.Listener, remote string) {
	defer close(it)
	p, err := pool.New(remote, pool.DefaultOpts())
	if err != nil {
		log.Fatal(err)
	}
	w := pool.Wrapper(p)
	defer w.Close()

	for {
		c, err := l.Accept()
		if err != nil {
			select {
			case <-is.Done():
				return
			default:
				log.Println(err)
				continue
			}
		}
		go func() {
			go func() {
				log.Println(io.Copy(c, w))
			}()
			log.Println(io.Copy(w, c))
		}()
	}
}

func main() {
	var listenAddr string
	var remoteAddr string

	flag.StringVar(&listenAddr, "listen", "0.0.0.0:9999", "listen addr (format: ip:port, like 0.0.0.0:9999)")
	flag.StringVar(&remoteAddr, "remote", "", "remote addr to be forwarded (format: ip:port, like 0.0.0.0:9999)")
	flag.Parse()
	if remoteAddr == "" {
		log.Fatal("invalid remote addr")
	}
	safeExit := make(chan struct{})
	isDone, done := context.WithCancel(context.Background())
	l, err := net.Listen("tcp4", listenAddr)
	if err != nil {
		log.Fatal(err)
	}

	go run(safeExit, isDone, l, remoteAddr)
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	done()
	l.Close()
	<-safeExit
}
