package warp

import (
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"syscall"
)

const SO_ORIGINAL_DST = 80

type Server struct {
	Addr string
	Port int
}

func (s *Server) Start() error {
	addr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%d", s.Addr, s.Port))
	if err != nil {
		return err
	}

	ln, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return err
	}
	defer ln.Close()
	log.Printf("warp listens to %s:%d", s.Addr, s.Port)

	for {
		conn, err := ln.AcceptTCP()
		if err != nil {
			log.Printf("accept error: %#v", err)
			continue
		}
		go s.HandleConnection(conn)
	}
}

func (s *Server) HandleConnection(conn *net.TCPConn) {
	log.Print("new connection")

	raddr, err := s.OriginalAddrDst(conn)
	if err != nil {
		log.Printf("original addr error: %#v", err)
		return
	}
	log.Printf("remote addr: %s origin addr: %s", conn.RemoteAddr(), raddr)

	laddr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:0", s.Addr))
	if err != nil {
		log.Printf("resolve tcp addr error: %#v", err)
		return
	}
	dstConn, err := net.DialTCP("tcp", laddr, raddr)
	if err != nil {
		log.Printf("dial '%s' error: %#v", raddr, err)
		return
	}

	p := &Pipe{Src: conn, Dst: dstConn}
	p.Do()
}

func (s *Server) OriginalAddrDst(conn *net.TCPConn) (*net.TCPAddr, error) {
	f, err := conn.File()
	if err != nil {
		return nil, err
	}
	defer f.Close()
	fd := f.Fd()

	addr, err := syscall.GetsockoptIPv6Mreq(int(fd), syscall.IPPROTO_IP, SO_ORIGINAL_DST)
	if err != nil {
		return nil, err
	}

	ip := strings.Join([]string{
		strconv.FormatUint(uint64(addr.Multiaddr[4]), 10),
		strconv.FormatUint(uint64(addr.Multiaddr[5]), 10),
		strconv.FormatUint(uint64(addr.Multiaddr[6]), 10),
		strconv.FormatUint(uint64(addr.Multiaddr[7]), 10),
	}, ".")
	port := uint16(addr.Multiaddr[2])<<8 + uint16(addr.Multiaddr[3])

	return &net.TCPAddr{
		IP:   net.ParseIP(ip),
		Port: int(port),
	}, nil
}
