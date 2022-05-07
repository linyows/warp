package warp

import (
	"fmt"
	"log"
	"net"
	"syscall"
	"time"
)

const SO_ORIGINAL_DST = 80

type Server struct {
	Addr  string
	Port  int
	Hooks []Hook
}

func (s *Server) Start() error {
	hooks, err := loadPlugins()
	if err != nil {
		return err
	}
	s.Hooks = hooks

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
		if s.Addr == fmt.Sprintf("%s", conn.RemoteAddr()) {
			conn.Close()
			log.Printf("closed connection due to same ip: %s", conn.RemoteAddr())
			continue
		}
		go s.HandleConnection(conn)
	}
}

func (s *Server) HandleConnection(conn net.Conn) {
	log.Printf("connected from %s", conn.RemoteAddr())

	raddr, err := s.OriginalAddrDst(conn)
	if err != nil {
		log.Printf("original addr error: %#v", err)
		return
	}

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

	p := &Pipe{
		id:    GenID().String(),
		sConn: conn,
		rConn: dstConn,
		rAddr: raddr,
	}
	p.afterCommHook = func(b Data, to Direction) {
		now := time.Now()
		log.Printf("[%s] %s %s %s", now.Format(TimeFormat), p.id, to, p.escapeCRLF(b))
		for _, hook := range s.Hooks {
			hook.AfterComm(&AfterCommData{
				ConnID:     p.id,
				OccurredAt: now,
				Data:       p.escapeCRLF(b),
				Direction:  to,
			})
		}
	}
	p.afterConnHook = func() {
		now := time.Now()
		log.Printf("[%s] %s from:%s to:%s", now.Format(TimeFormat), p.id, p.sMailAddr, p.rMailAddr)
		for _, hook := range s.Hooks {
			hook.AfterConn(&AfterConnData{
				ConnID:     p.id,
				OccurredAt: now,
				MailFrom:   p.sMailAddr,
				MailTo:     p.rMailAddr,
			})
		}
	}
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
