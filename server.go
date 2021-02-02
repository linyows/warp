package warp

import (
	"fmt"
	"log"
	"net"
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

	ln, err := net.Listen("tcp", addr.String())
	if err != nil {
		return err
	}
	defer ln.Close()
	log.Printf("warp listens to %s:%d", s.Addr, s.Port)

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("accept error: %#v", err)
			continue
		}
		go s.HandleConnection(conn)
	}
}

func (s *Server) HandleConnection(conn net.Conn) {
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
	dialer := &net.Dialer{LocalAddr: laddr}
	dstConn, err := dialer.Dial("tcp", raddr.String())
	if err != nil {
		log.Printf("dial '%s' error: %#v", raddr, err)
		return
	}

	p := &Pipe{sConn: conn, rConn: dstConn}
	p.Do()
}

func (s *Server) OriginalAddrDst(conn net.Conn) (*net.TCPAddr, error) {
	tcpConn, ok := conn.(*net.TCPConn)
	if !ok {
		return nil, fmt.Errorf("net.TCPConn cast error")
	}
	f, err := tcpConn.File()
	if err != nil {
		return nil, err
	}
	defer f.Close()
	fd := f.Fd()

	addr, err := syscall.GetsockoptIPv6Mreq(int(fd), syscall.IPPROTO_IP, SO_ORIGINAL_DST)
	if err != nil {
		return nil, err
	}

	ip := fmt.Sprintf("%d.%d.%d.%d", addr.Multiaddr[4],
		addr.Multiaddr[5], addr.Multiaddr[6], addr.Multiaddr[7])
	port := uint16(addr.Multiaddr[2])<<8 + uint16(addr.Multiaddr[3])

	return &net.TCPAddr{
		IP:   net.ParseIP(ip),
		Port: int(port),
	}, nil
}
