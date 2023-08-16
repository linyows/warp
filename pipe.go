package warp

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"regexp"
	"strings"
	"sync"
	"time"
)

type Pipe struct {
	id    string
	sConn net.Conn
	rConn net.Conn

	rAddr       *net.TCPAddr
	sMailAddr   []byte
	rMailAddr   []byte
	sServerName []byte
	rServerName []byte

	tls      bool
	readytls bool
	locked   bool
	blocker  chan interface{}

	timeAtConnected    time.Time
	timeAtDataStarting time.Time

	afterCommHook func(Data, Direction)
	afterConnHook func()
}

type Mediator func([]byte, int) ([]byte, int)
type Flow int
type Data []byte
type Direction string

const (
	mailFromPrefix string = "MAIL FROM:<"
	rcptToPrefix   string = "RCPT TO:<"
	mailRegex      string = `[A-z0-9.!#$%&'*+\-/=?^_\{|}~]{1,64}@[A-z0-9.\-]{1,255}`
	bufferSize     int    = 32 * 1024
	crlf           string = "\r\n"
	mailHeaderEnd  string = crlf + crlf

	srcToPxy Direction = ">|"
	pxyToDst Direction = "|>"
	dstToPxy Direction = "|<"
	//pxyToSrc Direction = "<|"
	srcToDst Direction = "->"
	dstToSrc Direction = "<-"
	onPxy    Direction = "--"

	upstream Flow = iota
	downstream

	// SMTP response codes
	codeServiceReady      int = 220
	codeStartingMailInput int = 354
	//codeActionCompleted int = 250
)

type ElapsedSeconds struct {
	Float64 float64
	Valid   bool
}

func (e ElapsedSeconds) String() string {
	if !e.Valid {
		return "nil"
	}
	return fmt.Sprintf("%dsec", int(e.Float64))
}

func (p *Pipe) Do() {
	p.timeAtConnected = time.Now()
	go p.afterCommHook([]byte(fmt.Sprintf("connected to %s", p.rAddr)), onPxy)

	var once sync.Once
	p.blocker = make(chan interface{})

	// Sender --- packet --> Proxy
	go func() {
		_, err := p.copy(upstream, func(b []byte, i int) ([]byte, int) {
			if !p.tls || p.rMailAddr == nil {
				p.pairing(b[0:i])
			}
			if !p.tls && p.readytls {
				p.locked = true
				er := p.starttls()
				if er != nil {
					go p.afterCommHook([]byte(fmt.Sprintf("starttls error: %s", er.Error())), pxyToDst)
				}
				p.readytls = false
				go p.afterCommHook(b[0:i], srcToPxy)
			}
			return b, i
		})
		if err != nil {
			go p.afterCommHook([]byte(fmt.Sprintf("io copy error: %s", err.Error())), pxyToDst)
		}
		once.Do(p.close())
	}()

	// Proxy <--- packet -- Receiver
	go func() {
		_, err := p.copy(downstream, func(b []byte, i int) ([]byte, int) {
			if p.isResponseOfEHLOWithStartTLS(b) {
				go p.afterCommHook(b[0:i], dstToPxy)
				b, i = p.removeStartTLSCommand(b, i)
			} else if p.isResponseOfReadyToStartTLS(b) {
				go p.afterCommHook(b[0:i], dstToPxy)
				er := p.connectTLS()
				if er != nil {
					go p.afterCommHook([]byte(fmt.Sprintf("TLS connection error: %s", er.Error())), dstToPxy)
				}
			}
			return b, i
		})
		if err != nil {
			go p.afterCommHook([]byte(fmt.Sprintf("io copy error: %s", err.Error())), dstToPxy)
		}
		once.Do(p.close())
	}()
}

func (p *Pipe) pairing(b []byte) {
	if bytes.Contains(b, []byte("HELO")) {
		p.sServerName = bytes.TrimSpace(bytes.Replace(b, []byte("HELO"), []byte(""), 1))
	}
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

func (p *Pipe) src(d Flow) net.Conn {
	if d == upstream {
		return p.sConn
	}
	return p.rConn
}

func (p *Pipe) dst(d Flow) net.Conn {
	if d == upstream {
		return p.rConn
	}
	return p.sConn
}

func (p *Pipe) copy(dr Flow, fn Mediator) (written int64, err error) {
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
		go p.afterCommHook([]byte(fmt.Sprintf("io.Reader size: %d", size)), onPxy)
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
				go p.afterCommHook(p.removeMailBody(buf[0:nr]), srcToDst)
			} else {
				// time before email input
				if fmt.Sprint(buf[:3]) == fmt.Sprint(codeStartingMailInput) {
					p.timeAtDataStarting = time.Now()
				}
				// remove buffering ready response
				if bytes.Contains(buf, []byte("Ready to start TLS")) || bytes.Contains(buf, []byte("SMTP server ready")) || bytes.Contains(buf, []byte("Start TLS")) {
					continue
				}
				go p.afterCommHook(buf[0:nr], dstToSrc)
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
	go p.afterCommHook([]byte(cmd), pxyToDst)
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
	go p.afterCommHook(buf[0:i], dstToPxy)
	return nil
}

func (p *Pipe) waitForTLSConn(b []byte, i int) {
	go p.afterCommHook([]byte("pipe locked for tls connection"), onPxy)
	<-p.blocker
	go p.afterCommHook([]byte("tls connected, to pipe unlocked"), onPxy)
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
		defer p.afterConnHook()
		defer p.afterCommHook([]byte("connections closed"), onPxy)
		p.rConn.Close()
		p.sConn.Close()
	}
}

func (p *Pipe) isResponseOfEHLOWithStartTLS(b []byte) bool {
	return !p.tls && !p.locked && bytes.Contains(b, []byte("STARTTLS"))
}

func (p *Pipe) isResponseOfReadyToStartTLS(b []byte) bool {
	return !p.tls && p.locked && bytes.Contains(b, []byte(fmt.Sprint(codeServiceReady)))
}

func (p *Pipe) removeMailBody(b Data) Data {
	i := bytes.Index(b, []byte(mailHeaderEnd))
	if i == -1 {
		return b
	}
	return b[:i]
}

func (p *Pipe) removeStartTLSCommand(b []byte, i int) ([]byte, int) {
	lastLine := "250 STARTTLS" + crlf
	intermediateLine := "250-STARTTLS" + crlf

	if bytes.Contains(b, []byte(lastLine)) {
		old := []byte(lastLine)
		b = bytes.Replace(b, old, []byte(""), 1)
		i = i - len(old)
		p.readytls = true

		arr := strings.Split(string(b), crlf)
		num := len(arr) - 2
		arr[num] = strings.Replace(arr[num], "250-", "250 ", 1)
		b = []byte(strings.Join(arr, crlf))

	} else if bytes.Contains(b, []byte(intermediateLine)) {
		old := []byte(intermediateLine)
		b = bytes.Replace(b, old, []byte(""), 1)
		i = i - len(old)
		p.readytls = true

	} else {
		go p.afterCommHook([]byte("starttls replace error"), dstToPxy)
	}

	return b, i
}

func (p *Pipe) elapsedSeconds() ElapsedSeconds {
	if p.timeAtDataStarting.IsZero() {
		log.Print("time at data starting is zero")
		return ElapsedSeconds{Valid: false}
	}
	if p.timeAtConnected.IsZero() {
		log.Print("time at connected is zero")
		return ElapsedSeconds{Valid: false}
	}
	return ElapsedSeconds{
		Float64: p.timeAtDataStarting.Sub(p.timeAtConnected).Seconds(),
		Valid:   true,
	}
}
