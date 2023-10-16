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
	Addr             string
	Port             int
	Hooks            []Hook
	OutboundAddr     string
	Verbose          bool
	log              *log.Logger
	MessageSizeLimit int
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
	if s.MessageSizeLimit == 0 {
		// default is around 10MB (https://www.postfix.org/postconf.5.html)
		s.MessageSizeLimit = 10240000
	}

	pl := &Plugins{}
	if err := pl.load(); err != nil {
		return err
	}
	s.Hooks = append(s.Hooks, pl.hooks...)
	for _, hook := range s.Hooks {
		s.log.Printf("use %s hook", hook.Name())
		hook.AfterInit()
	}

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

func (s *Server) HandleConnection(conn net.Conn) {
	uuid := GenID().String()
	if s.Verbose {
		s.log.Printf("%s %s connected from %s", uuid, onPxy, conn.RemoteAddr())
	}

	raddr, err := s.OriginalAddrDst(conn)
	if err != nil {
		s.log.Printf("%s %s original addr error: %s(%#v)", uuid, onPxy, err.Error(), err)
		return
	}

	go func() {
		now := time.Now()
		b := []byte(fmt.Sprintf("connecting to %s", raddr))
		if s.Verbose {
			s.log.Printf("%s %s %s", uuid, onPxy, b)
		}
		for _, hook := range s.Hooks {
			hook.AfterComm(&AfterCommData{
				ConnID:     uuid,
				OccurredAt: now,
				Data:       b,
				Direction:  onPxy,
			})
		}
	}()

	laddr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:0", s.OutboundAddr))
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
		id:         uuid,
		sConn:      conn,
		rConn:      dstConn,
		rAddr:      raddr,
		bufferSize: s.MessageSizeLimit,
	}
	p.afterCommHook = func(b Data, to Direction) {
		now := time.Now()
		if s.Verbose {
			s.log.Printf("%s %s %s", p.id, to, p.escapeCRLF(b))
		}
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
		sM := p.sMailAddr
		rM := p.rMailAddr
		if len(sM) == 0 {
			sM = []byte("unknown")
		}
		if len(rM) == 0 {
			rM = []byte("unknown")
		}

		now := time.Now()
		elapse := p.elapse()
		if s.Verbose {
			b := fmt.Sprintf("from:%s to:%s elapse:%s", sM, rM, elapse)
			s.log.Printf("%s %s %s", p.id, onPxy, b)
		}
		for _, hook := range s.Hooks {
			hook.AfterConn(&AfterConnData{
				ConnID:     p.id,
				OccurredAt: now,
				MailFrom:   sM,
				MailTo:     rM,
				Elapse:     elapse,
			})
		}
	}

	p.Do()
	p.Close()
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
