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
	mailFromPrefix       string = "MAIL FROM:<"
	rcptToPrefix         string = "RCPT TO:<"
	mailRegex            string = `(?i)MAIL\s+FROM\s*:\s*<[A-z0-9.!#$%&'*+\-/=?^_\{|}~]{1,64}@[A-z0-9.\-]{1,255}>`
	rcptToRegex          string = `(?i)RCPT\s+TO\s*:\s*<[A-z0-9.!#$%&'*+\-/=?^_\{|}~]{1,64}@[A-z0-9.\-]{1,255}>`
	mailRegexStrict      string = `(?i)MAIL FROM:<[A-z0-9.!#$%&'*+\-/=?^_\{|}~]{1,64}@[A-z0-9.\-]{1,255}>`
	rcptToRegexStrict    string = `(?i)RCPT TO:<[A-z0-9.!#$%&'*+\-/=?^_\{|}~]{1,64}@[A-z0-9.\-]{1,255}>`
	crlf                 string = "\r\n"
	mailHeaderEnd        string = crlf + crlf

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
	mailFromRegex       = regexp.MustCompile(mailRegex)
	mailToRegex         = regexp.MustCompile(rcptToRegex)
	mailFromRegexStrict = regexp.MustCompile(mailRegexStrict)
	mailToRegexStrict   = regexp.MustCompile(rcptToRegexStrict)
)

func (e Elapse) String() string {
	return fmt.Sprintf("%d msec", e)
}

// toLower converts ASCII byte to lowercase
func toLower(b byte) byte {
	if b >= 'A' && b <= 'Z' {
		return b + ('a' - 'A')
	}
	return b
}

// containsFold performs case-insensitive bytes.Contains for ASCII
func containsFold(s, substr []byte) bool {
	if len(substr) == 0 {
		return true
	}
	if len(substr) > len(s) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if toLower(s[i+j]) != toLower(substr[j]) {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

// isRFCCompliant checks if the matched command strictly follows RFC 5321 syntax
// RFC 5321 Section 3.3: "spaces are not permitted on either side of the colon
// following FROM in the MAIL command or TO in the RCPT command"
func isRFCCompliant(match []byte, strictRegex *regexp.Regexp) bool {
	return strictRegex.Match(match)
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
	if containsFold(b, []byte("HELO ")) {
		// Find the position of HELO (case-insensitive) and extract the hostname
		upper := bytes.ToUpper(b)
		idx := bytes.Index(upper, []byte("HELO "))
		if idx >= 0 {
			p.sServerName = bytes.TrimSpace(b[idx+5:])
		}
	}
	if containsFold(b, []byte("EHLO ")) {
		// Find the position of EHLO (case-insensitive) and extract the hostname
		upper := bytes.ToUpper(b)
		idx := bytes.Index(upper, []byte("EHLO "))
		if idx >= 0 {
			p.sServerName = bytes.TrimSpace(b[idx+5:])
		}
	}
}

func (p *Pipe) setSenderMailAddress(b []byte) {
	match := mailFromRegex.Find(b)
	if match != nil {
		// Extract email address from "MAIL FROM:<email>" (case-insensitive, relaxed spacing)
		// Find the position of '<' and '>'
		start := bytes.IndexByte(match, '<')
		end := bytes.IndexByte(match, '>')
		if start >= 0 && end > start {
			p.sMailAddr = match[start+1 : end]

			// Check RFC 5321 compliance
			if !isRFCCompliant(match, mailFromRegexStrict) {
				go p.afterCommHook([]byte(fmt.Sprintf("RFC 5321 violation: %q (spaces not permitted around colon)", match)), onPxy)
			}
		}
	}
}

func (p *Pipe) setReceiverMailAddressAndServerName(b []byte) {
	match := mailToRegex.Find(b)
	if match != nil {
		// Extract email address from "RCPT TO:<email>" (case-insensitive, relaxed spacing)
		// Find the position of '<' and '>'
		start := bytes.IndexByte(match, '<')
		end := bytes.IndexByte(match, '>')
		if start >= 0 && end > start {
			p.rMailAddr = match[start+1 : end]
			parts := bytes.Split(p.rMailAddr, []byte("@"))
			if len(parts) == 2 {
				p.rServerName = parts[1]
			}

			// Check RFC 5321 compliance
			if !isRFCCompliant(match, mailToRegexStrict) {
				go p.afterCommHook([]byte(fmt.Sprintf("RFC 5321 violation: %q (spaces not permitted around colon)", match)), onPxy)
			}
		}
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
