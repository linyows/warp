package warp

import (
	"bufio"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/smtp"
	"os"
	"strings"
	"time"
)

const (
	outgoing  Direction = "->"
	incomming Direction = "<-"
)

func WaitForServerListen(ip string, port int) {
	host := fmt.Sprintf("%s:%d", ip, port)
	fmt.Printf("Wait for port %d listen...", port)
	for {
		timeout := time.Second
		conn, err := net.DialTimeout("tcp", host, timeout)
		if err != nil {
			fmt.Print(".")
		}
		if conn != nil {
			fmt.Print("\n")
			conn.Close()
			break
		}
	}
}

type SMTPClient struct {
	IP   string
	Port int
}

func (c *SMTPClient) SendEmail() error {
	s, err := smtp.Dial(fmt.Sprintf("%s:%d", c.IP, c.Port))
	if err != nil {
		return fmt.Errorf("smtp.Dial(%s:%d): %#v", c.IP, c.Port, err)
	}
	if err := s.Mail("alice@example.test"); err != nil {
		return fmt.Errorf("smtp mail error: %#v", err)
	}
	if err := s.Rcpt("bob@example.local"); err != nil {
		return fmt.Errorf("smtp rcpt error: %#v", err)
	}
	wc, err := s.Data()
	if err != nil {
		return fmt.Errorf("smtp data error: %#v", err)
	}
	_, err = fmt.Fprintf(wc, "This is the email body")
	if err != nil {
		return fmt.Errorf("smtp data print error: %#v", err)
	}
	if err = wc.Close(); err != nil {
		return fmt.Errorf("smtp close print error: %#v", err)
	}
	if err = s.Quit(); err != nil {
		return fmt.Errorf("smtp quit error: %#v", err)
	}
	return nil
}

type SMTPServer struct {
	IP       string
	Port     int
	Hostname string
	log      *log.Logger
}

func (s *SMTPServer) Serve() error {
	if s.log == nil {
		s.log = log.New(os.Stderr, "", log.LstdFlags)
	}

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", s.Port))
	if err != nil {
		return fmt.Errorf("net.Listen(tcp) error: %#v", err)
	}
	defer listener.Close()

	s.log.Printf("SMTP server is listening on :%d\n", s.Port)

	for {
		conn, err := listener.Accept()
		if err != nil {
			s.log.Println("Accept error:", err)
			continue
		}
		c := &SMTPConn{hostname: s.Hostname, id: GenID().String(), log: s.log}
		go c.handle(conn)
	}
}

type SMTPConn struct {
	reader   *bufio.Reader
	writer   *bufio.Writer
	id       string
	hostname string
	data     bool
	log      *log.Logger
}

func (c *SMTPConn) handle(conn net.Conn) {
	defer conn.Close()

	c.reader = bufio.NewReader(conn)
	c.writer = bufio.NewWriter(conn)

	c.writeStringWithLog(fmt.Sprintf("220 %s ESMTP Server", c.hostname))
	c.writer.Flush()
	c.data = false

	for {
		line, err := c.reader.ReadString('\n')
		if err != nil {
			if !errors.Is(err, io.EOF) {
				c.log.Printf("%s %s conn ReadString error: %#v", c.id, "--", err)
			}
			return
		}

		c.log.Printf("%s %s %s", c.id, incomming, line)
		line = strings.TrimSpace(line)
		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}

		commands := strings.Split(line, crlf)
		first := ""
		second := ""

		for _, v := range commands {
			parts := strings.Fields(strings.TrimSpace(v))
			if len(parts) == 0 {
				continue
			}
			first = strings.ToUpper(parts[0])
			if len(parts) > 1 {
				second = strings.ToUpper(parts[1])
			}

			switch first {
			case "EHLO":
				c.writeStringWithLog(fmt.Sprintf("250-%s\r\n250-PIPELINING\r\n250-SIZE 10240000\r\n250-STARTTLS\r\n250 8BITMIME", c.hostname))
			case "HELO":
				c.writeStringWithLog(fmt.Sprintf("250 Hello %s", parts[1]))
			case "MAIL":
				if strings.Contains(second, "FROM:") {
					c.writeStringWithLog("250 2.1.0 Ok")
				}
			case "RCPT":
				if strings.Contains(second, "TO:") {
					c.writeStringWithLog("250 2.1.5 Ok")
				}
			case "DATA":
				c.data = true
				c.writeStringWithLog("354 End data with <CR><LF>.<CR><LF>")
			case "QUIT":
				c.writeStringWithLog("221 2.0.0 Bye")
				c.writer.Flush()
				return
			case ".":
				c.data = false
				c.writeStringWithLog("250 2.0.0 Ok: queued")
			case "RSET":
				c.writeStringWithLog("250 2.0.0 Ok")
			case "NOOP":
				c.writeStringWithLog("250 2.0.0 Ok")
			case "VRFY":
				c.writeStringWithLog("502 5.5.1 VRFY command is disabled")
			case "STARTTLS":
				c.writeStringWithLog("220 2.0.0 Ready to start TLS")
				c.writer.Flush()
				c.startTLS(conn)
			default:
				if !c.data {
					c.writeStringWithLog("500 Command not recognized")
				}
			}
		}

		c.writer.Flush()
	}
}

func (c *SMTPConn) writeStringWithLog(str string) {
	_, err := c.writer.WriteString(str + crlf)
	if err != nil {
		c.log.Printf("%s %s WriteString error: %#v", c.id, outgoing, err)
	}
	c.log.Printf("%s %s %s", c.id, outgoing, strings.ReplaceAll(str, crlf, "\\r\\n"))
}

func (c *SMTPConn) startTLS(conn net.Conn) {
	cert, err := tls.LoadX509KeyPair("testdata/server.crt", "testdata/server.key")
	if err != nil {
		c.log.Printf("%s %s Error loading server certificate: %#v", c.id, "--", err)
		return
	}
	tlsConfig := &tls.Config{Certificates: []tls.Certificate{cert}}
	tlsConn := tls.Server(conn, tlsConfig)
	if err := tlsConn.Handshake(); err != nil {
		c.writeStringWithLog("550 5.0.0 Handshake error")
		return
	}
	c.reader = bufio.NewReader(tlsConn)
	c.writer = bufio.NewWriter(tlsConn)
}
