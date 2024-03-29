package warp

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"regexp"
	"strings"
	"time"
)

type Pipe struct {
	id         string
	sConn      net.Conn
	rConn      net.Conn
	bufferSize int

	rAddr       *net.TCPAddr
	sMailAddr   []byte
	rMailAddr   []byte
	sServerName []byte
	rServerName []byte

	tls      bool
	readytls bool
	locked   bool
	blocker  chan interface{}

	isWaitedStarttlsRes bool
	isHeaderRemoved     bool

	timeAtConnected    time.Time
	timeAtDataStarting time.Time

	afterCommHook func(Data, Direction)
	afterConnHook func()
}

type Mediator func([]byte, int) ([]byte, int, bool)
type Flow int
type Data []byte
type Direction string
type Elapse int

const (
	mailFromPrefix string = "MAIL FROM:<"
	rcptToPrefix   string = "RCPT TO:<"
	mailRegex      string = `[A-z0-9.!#$%&'*+\-/=?^_\{|}~]{1,64}@[A-z0-9.\-]{1,255}`
	crlf           string = "\r\n"
	mailHeaderEnd  string = crlf + crlf

	srcToPxy Direction = ">|"
	pxyToDst Direction = "|>"
	dstToPxy Direction = "|<"
	pxyToSrc Direction = "<|"
	srcToDst Direction = "->"
	dstToSrc Direction = "<-"
	onPxy    Direction = "--"

	upstream Flow = iota
	downstream

	// SMTP response codes
	codeServiceReady      int = 220
	codeStartingMailInput int = 354
	codeActionCompleted   int = 250
)

var (
	mailFromRegex = regexp.MustCompile(mailFromPrefix + mailRegex)
	mailToRegex   = regexp.MustCompile(rcptToPrefix + mailRegex)
)

func (e Elapse) String() string {
	return fmt.Sprintf("%d msec", e)
}

func (p *Pipe) mediateOnUpstream(b []byte, i int) ([]byte, int, bool) {
	data := b[0:i]

	if !p.tls || p.rMailAddr == nil {
		p.setSenderMailAddress(data)
		p.setSenderServerName(data)
		p.setReceiverMailAddressAndServerName(data)
	}

	if !p.tls && p.readytls {
		p.locked = true
		er := p.starttls()
		p.isWaitedStarttlsRes = true
		if er != nil {
			go p.afterCommHook([]byte(fmt.Sprintf("starttls error: %s", er.Error())), pxyToDst)
		}
		p.readytls = false
		go p.afterCommHook(data, srcToPxy)
	}

	if p.locked {
		p.waitForTLSConn(b, i)
		go p.afterCommHook(data, pxyToDst)
	} else {
		if !p.isHeaderRemoved {
			go p.afterCommHook(p.removeMailBody(data), srcToDst)
		}
	}

	return b, i, false
}

func (p *Pipe) mediateOnDownstream(b []byte, i int) ([]byte, int, bool) {
	data := b[0:i]

	if p.isResponseOfEHLOWithStartTLS(b) {
		go p.afterCommHook(data, dstToPxy)
		b, i = p.removeStartTLSCommand(b, i)
		data = b[0:i]
	} else if p.isResponseOfReadyToStartTLS(b) {
		go p.afterCommHook(data, dstToPxy)
		er := p.connectTLS()
		if er != nil {
			go p.afterCommHook([]byte(fmt.Sprintf("TLS connection error: %s", er.Error())), dstToPxy)
		}
	}

	// remove buffering "220 2.0.0 Ready to start TLS" response
	if p.isWaitedStarttlsRes {
		p.isWaitedStarttlsRes = false
		return b, i, true
	}

	// time before email input
	p.setTimeAtDataStarting(b)

	if p.isResponseOfEHLOWithoutStartTLS(b) {
		go p.afterCommHook(data, pxyToSrc)
	} else {
		go p.afterCommHook(data, dstToSrc)
	}

	return b, i, false
}

func (p *Pipe) setTimeAtDataStarting(b []byte) {
	list := bytes.Split(b, []byte(crlf))
	for _, v := range list {
		if len(v) >= 3 && string(v[:3]) == fmt.Sprint(codeStartingMailInput) {
			p.timeAtDataStarting = time.Now()
		}
	}
}

func (p *Pipe) Do() {
	p.timeAtConnected = time.Now()
	go p.afterCommHook([]byte(fmt.Sprintf("connected to %s", p.rAddr)), onPxy)

	p.blocker = make(chan interface{})
	done := make(chan bool)

	// Sender --- packet --> Proxy
	go func() {
		_, err := p.copy(upstream, p.mediateOnUpstream)
		if err != nil && !errors.Is(err, net.ErrClosed) {
			go p.afterCommHook([]byte(fmt.Sprintf("io copy error: %s", err)), pxyToDst)
		}
		done <- true
	}()

	// Proxy <--- packet -- Receiver
	go func() {
		_, err := p.copy(downstream, p.mediateOnDownstream)
		if err != nil && !errors.Is(err, net.ErrClosed) {
			go p.afterCommHook([]byte(fmt.Sprintf("io copy error: %s", err)), dstToPxy)
		}
		done <- true
	}()

	<-done
}

func (p *Pipe) setSenderServerName(b []byte) {
	if bytes.Contains(b, []byte("HELO")) {
		p.sServerName = bytes.TrimSpace(bytes.Replace(b, []byte("HELO"), []byte(""), 1))
	}
	if bytes.Contains(b, []byte("EHLO")) {
		p.sServerName = bytes.TrimSpace(bytes.Replace(b, []byte("EHLO"), []byte(""), 1))
	}
}

func (p *Pipe) setSenderMailAddress(b []byte) {
	if bytes.Contains(b, []byte(mailFromPrefix)) {
		p.sMailAddr = bytes.Replace(mailFromRegex.Find(b), []byte(mailFromPrefix), []byte(""), 1)
	}
}

func (p *Pipe) setReceiverMailAddressAndServerName(b []byte) {
	if bytes.Contains(b, []byte(rcptToPrefix)) {
		p.rMailAddr = bytes.Replace(mailToRegex.Find(b), []byte(rcptToPrefix), []byte(""), 1)
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
	size := p.bufferSize
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
	buf := make([]byte, p.bufferSize)

	for {
		var isContinue bool
		if p.locked {
			continue
		}

		nr, er := p.src(dr).Read(buf)
		if nr > 0 {
			// Run the Mediator!
			buf, nr, isContinue = fn(buf, nr)
			if nr == 0 || isContinue {
				continue
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
	buf := make([]byte, 64*1024)
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

func (p *Pipe) Close() {
	p.rConn.Close()
	p.sConn.Close()
	go p.afterCommHook([]byte("connections closed"), onPxy)
	go p.afterConnHook()
}

func (p *Pipe) isResponseOfEHLOWithStartTLS(b []byte) bool {
	return !p.tls && !p.locked && bytes.Contains(b, []byte(fmt.Sprint(codeActionCompleted))) && bytes.Contains(b, []byte("STARTTLS"))
}

func (p *Pipe) isResponseOfEHLOWithoutStartTLS(b []byte) bool {
	return !p.tls && !p.locked && bytes.Contains(b, []byte(fmt.Sprint(codeActionCompleted))) && !bytes.Contains(b, []byte("STARTTLS"))
}

func (p *Pipe) isResponseOfReadyToStartTLS(b []byte) bool {
	return !p.tls && p.locked && bytes.Contains(b, []byte(fmt.Sprint(codeServiceReady)))
}

func (p *Pipe) removeMailBody(b Data) Data {
	i := bytes.Index(b, []byte(mailHeaderEnd))
	if i == -1 {
		return b
	}
	p.isHeaderRemoved = true
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

func (p *Pipe) elapse() Elapse {
	if p.timeAtConnected.IsZero() {
		return -1
	}
	if p.timeAtDataStarting.IsZero() {
		return -2
	}
	return Elapse(p.timeAtDataStarting.Sub(p.timeAtConnected).Milliseconds())
}
