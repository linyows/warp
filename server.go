package warp

import (
	"fmt"
	"log"
	"net"
	"os"
	"syscall"
	"time"
)

const SO_ORIGINAL_DST = 80

type Server struct {
	Addr          string
	Port          int
	Hooks         []Hook
	OutboundAddr  string
	OutboundPorts *PortRange
	log           *log.Logger
}

// These are global variables for integration test.
var (
	specifiedDstIP   = ""
	specifiedDstPort = 0
)

func (s *Server) Start() error {
	if s.log == nil {
		s.log = log.New(os.Stderr, "", log.Ldate|log.Ltime|log.Lmicroseconds)
	}

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
	s.log.Printf("warp listens to %s:%d", s.Addr, s.Port)

	for {
		conn, err := ln.Accept()
		if err != nil {
			s.log.Printf("accept error(is the warp port open globally?): %s(%#v)", err.Error(), err)
			continue
		}
		if s.Addr == conn.RemoteAddr().String() {
			conn.Close()
			s.log.Printf("closed connection due to same ip(looping requests to warp?): %s", conn.RemoteAddr())
			continue
		}
		go s.HandleConnection(conn)
	}
}

func (s *Server) getOutboundAddrAndPort() (string, int) {
	port := 0
	addr := s.Addr

	if s.OutboundPorts != nil {
		port, _ = s.OutboundPorts.TakeOut()
	}
	if s.OutboundAddr != "" {
		addr = s.OutboundAddr
	}

	return addr, port
}

func (s *Server) HandleConnection(conn net.Conn) {
	uuid := GenID().String()
	s.log.Printf("%s %s connected from %s", uuid, onPxy, conn.RemoteAddr())

	raddr, err := s.OriginalAddrDst(conn)
	if err != nil {
		s.log.Printf("%s %s original addr error: %s(%#v)", uuid, onPxy, err.Error(), err)
		return
	}

	go func() {
		now := time.Now()
		b := []byte(fmt.Sprintf("connecting to %s", raddr))
		s.log.Printf("%s %s %s", uuid, onPxy, b)
		for _, hook := range s.Hooks {
			hook.AfterComm(&AfterCommData{
				ConnID:     uuid,
				OccurredAt: now,
				Data:       b,
				Direction:  onPxy,
			})
		}
	}()

	oAddr, oPort := s.getOutboundAddrAndPort()
	laddr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%d", oAddr, oPort))
	if err != nil {
		s.log.Printf("%s %s resolve tcp addr error: %s(%#v)", uuid, onPxy, err.Error(), err)
		return
	}
	dialer := &net.Dialer{LocalAddr: laddr}
	dstConn, err := dialer.Dial("tcp", raddr.String())
	if err != nil {
		s.log.Printf("%s %s dial `%s` with `%s` error: %s(%#v)", uuid, onPxy, raddr, laddr, err.Error(), err)
		return
	}

	p := &Pipe{
		id:    uuid,
		sConn: conn,
		rConn: dstConn,
		rAddr: raddr,
	}
	p.afterCommHook = func(b Data, to Direction) {
		now := time.Now()
		s.log.Printf("%s %s %s", p.id, to, p.escapeCRLF(b))
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
		elapse := p.elapse()
		b := fmt.Sprintf("from:%s to:%s elapse:%s", p.sMailAddr, p.rMailAddr, elapse)
		s.log.Printf("%s %s %s", p.id, onPxy, b)
		for _, hook := range s.Hooks {
			hook.AfterConn(&AfterConnData{
				ConnID:     p.id,
				OccurredAt: now,
				MailFrom:   p.sMailAddr,
				MailTo:     p.rMailAddr,
				Elapse:     elapse,
			})
		}
	}
	p.Do()
}

func (s *Server) OriginalAddrDst(conn net.Conn) (*net.TCPAddr, error) {
	if specifiedDstIP != "" && specifiedDstPort != 0 {
		return &net.TCPAddr{
			IP:   net.ParseIP(specifiedDstIP),
			Port: specifiedDstPort,
		}, nil
	}

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
