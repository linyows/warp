package warp

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"regexp"
	"sync"
)

type Pipe struct {
	sConn       net.Conn
	rConn       net.Conn
	sMailAddr   []byte
	rMailAddr   []byte
	sServerName []byte
	rServerName []byte
	tls         bool
	readytls    bool
	locked      bool
	blocker     chan interface{}
}

type Mediator func([]byte, int) ([]byte, int)
type Direction int

const (
	mailFromPrefix  string    = "MAIL FROM:<"
	rcptToPrefix    string    = "RCPT TO:<"
	mailRegex       string    = `[+A-z0-9.-]+@[A-z0-9.-]+`
	bufferSize      int       = 32 * 1024
	readyToStartTLS string    = "Ready to start TLS"
	crlf            string    = "\r\n"
	upstream        Direction = iota
	downstream
)

func (p *Pipe) Do() {
	var once sync.Once
	p.blocker = make(chan interface{})

	go func() {
		_, err := p.copy(upstream, func(b []byte, i int) ([]byte, int) {
			if !p.tls {
				p.pairing(b[0:i])
			}
			if !p.tls && p.readytls {
				p.locked = true
				er := p.starttls()
				if er != nil {
					log.Printf("upstream starttls error: %s", er.Error())
				}
				p.readytls = false
				log.Printf(">| %s", p.escapeCRLF(b[0:i]))
			}
			return b, i
		})
		if err != nil {
			log.Printf("upstream copy error: %s", err.Error())
		}
		once.Do(p.close())
	}()

	go func() {
		_, err := p.copy(downstream, func(b []byte, i int) ([]byte, int) {
			if !p.tls && bytes.Contains(b, []byte("STARTTLS")) {
				log.Printf("|< %s", p.escapeCRLF(b[0:i]))
				old := []byte("250-STARTTLS\r\n")
				b = bytes.Replace(b, old, []byte(""), 1)
				i = i - len(old)
				p.readytls = true
			} else if !p.tls && bytes.Contains(b, []byte(readyToStartTLS)) {
				log.Printf("|< %s", p.escapeCRLF(b[0:i]))
				er := p.connectTLS()
				if er != nil {
					log.Printf("downstream connectTLS error: %s", er.Error())
				}
			}
			return b, i
		})
		if err != nil {
			log.Printf("downstream copy error: %s", err.Error())
		}
		once.Do(p.close())
	}()
}

func (p *Pipe) pairing(b []byte) {
	if bytes.Contains(b, []byte("EHLO")) {
		p.sServerName = bytes.TrimSpace(bytes.Replace(b, []byte("EHLO"), []byte(""), 1))
	}
	if bytes.Contains(b, []byte(mailFromPrefix)) {
		re := regexp.MustCompile(mailFromPrefix + mailRegex)
		p.sMailAddr = bytes.Replace(re.Find(b), []byte(mailFromPrefix), []byte(""), 1)
	}
	if bytes.Contains(b, []byte(rcptToPrefix)) {
		re := regexp.MustCompile(rcptToPrefix + mailRegex)
		p.rMailAddr = bytes.Replace(re.Find(b), []byte(rcptToPrefix), []byte(""), 1)
		p.rServerName = bytes.Split(p.rMailAddr, []byte("@"))[1]
	}
}

func (p *Pipe) src(d Direction) net.Conn {
	if d == upstream {
		return p.sConn
	}
	return p.rConn
}

func (p *Pipe) dst(d Direction) net.Conn {
	if d == upstream {
		return p.rConn
	}
	return p.sConn
}

func (p *Pipe) copy(dr Direction, fn Mediator) (written int64, err error) {
	size := bufferSize
	src, ok := p.src(dr).(io.Reader)
	if !ok {
		err = fmt.Errorf("io.Reader cast error")
	}
	if l, ok := src.(*io.LimitedReader); ok && int64(size) > l.N {
		if l.N < 1 {
			size = 1
		} else {
			size = int(l.N)
		}
	}
	buf := make([]byte, bufferSize)

	for {
		if p.locked {
			continue
		}

		nr, er := p.src(dr).Read(buf)
		if nr > 0 {
			buf, nr = fn(buf, nr)
			if dr == upstream && p.locked {
				p.waitForTLSConn(buf, nr)
			}
			if nr == 0 {
				continue
			}
			if dr == upstream {
				log.Printf("-> %s", p.escapeCRLF(buf[0:nr]))
			} else {
				if bytes.Contains(buf, []byte(readyToStartTLS)) {
					continue
				}
				log.Printf("<- %s", p.escapeCRLF(buf[0:nr]))
			}
			nw, ew := p.dst(dr).Write(buf[0:nr])
			if nw > 0 {
				written += int64(nw)
			}
			if ew != nil {
				err = ew
				break
			}
			if nr != nw {
				err = io.ErrShortWrite
				break
			}
		}
		if er != nil {
			if er != io.EOF {
				err = er
			}
			break
		}
	}

	return written, err
}

func (p *Pipe) cmd(format string, args ...interface{}) error {
	cmd := fmt.Sprintf(format+crlf, args...)
	log.Printf("|> %s", p.escapeCRLF([]byte(cmd)))
	_, err := p.rConn.Write([]byte(cmd))
	if err != nil {
		return err
	}
	return nil
}

func (p *Pipe) ehlo() error {
	return p.cmd("EHLO %s", p.sServerName)
}

func (p *Pipe) starttls() error {
	return p.cmd("STARTTLS")
}

func (p *Pipe) readReceiverConn() error {
	buf := make([]byte, bufferSize)
	i, err := p.rConn.Read(buf)
	if err != nil {
		return err
	}
	log.Printf("|< %s", p.escapeCRLF(buf[0:i]))
	return nil
}

func (p *Pipe) waitForTLSConn(b []byte, i int) {
	log.Print("pipe locked for tls connection")
	<-p.blocker
	log.Print("tls connected, to pipe unlocked")
	p.locked = false
}

func (p *Pipe) connectTLS() error {
	p.rConn = tls.Client(p.rConn, &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         string(p.rServerName),
	})

	err := p.ehlo()
	if err != nil {
		return err
	}

	err = p.readReceiverConn()
	if err != nil {
		return err
	}

	p.tls = true
	p.blocker <- false

	return nil
}

func (p *Pipe) escapeCRLF(b []byte) []byte {
	return bytes.ReplaceAll(b, []byte(crlf), []byte("\\r\\n"))
}

func (p *Pipe) close() func() {
	return func() {
		defer p.sConn.Close()
		defer p.rConn.Close()
		defer log.Print("connections closed")
	}
}
