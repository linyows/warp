package warp

import (
	"fmt"
	"log"
	"net"
	"os"
	"syscall"
	"unsafe"
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

	raddr, err := s.OriginalAddr(conn)
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

func (s *Server) OriginalAddr(conn *net.TCPConn) (*net.TCPAddr, error) {
	f, err := conn.File()
	if err != nil {
		return nil, err
	}
	defer f.Close()

	fd := int(f.Fd())
	if err = syscall.SetNonblock(fd, true); err != nil {
		return nil, os.NewSyscallError("setnonblock", err)
	}

	var addr syscall.RawSockaddrInet4
	var len uint32
	len = uint32(unsafe.Sizeof(addr))
	err = getsockopt(fd, syscall.IPPROTO_IP, SO_ORIGINAL_DST, unsafe.Pointer(&addr), &len)

	if err != nil {
		return nil, os.NewSyscallError("getsockopt", err)
	}

	ip := make([]byte, 4)
	for i, b := range addr.Addr {
		ip[i] = b
	}
	pb := *(*[2]byte)(unsafe.Pointer(&addr.Port))

	return &net.TCPAddr{
		IP:   ip,
		Port: int(pb[0])*256 + int(pb[1]),
	}, nil
}

func getsockopt(s int, level int, optname int, optval unsafe.Pointer, optlen *uint32) (err error) {
	_, _, e := syscall.Syscall6(
		syscall.SYS_GETSOCKOPT, uintptr(s), uintptr(level), uintptr(optname),
		uintptr(optval), uintptr(unsafe.Pointer(optlen)), 0)
	if e != 0 {
		return e
	}
	return
}
