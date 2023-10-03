package warp

import (
	"net"
	"strconv"
	"testing"
)

func listenLocalPort(t *testing.T) (net.Listener, int) {
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		for {
			conn, _ := ln.Accept()
			if conn != nil {
				conn.Close()
			}
		}
	}()

	_, p, err := net.SplitHostPort(ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}

	port, err := strconv.Atoi(p)
	if err != nil {
		t.Fatal(err)
	}

	return ln, port
}

func TestTakeOut(t *testing.T) {
	ip := "127.0.0.1"

	ln, port := listenLocalPort(t)
	defer ln.Close()

	start := port - 10
	end := port + 10

	r1 := &PortRange{start: start, end: end}
	got1, err := r1.TakeOut(ip)
	if start != got1 || err != nil {
		t.Errorf("port range take out expected %d, but got %d", start, got1)
	}

	r2 := &PortRange{start: port, end: port}
	got2, err := r2.TakeOut(ip)
	if 0 != got2 || err == nil {
		t.Error("port range take out expected error, but got no error")
	}
}
