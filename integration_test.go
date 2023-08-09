package warp

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"net/smtp"
	"os"
	"strings"
	"testing"
	"time"
)

const (
	ip       string = "127.0.0.1"
	port     string = "10025"
	hostname string = "example.local"
)

type SMTPConnection struct {
	reader *bufio.Reader
	writer *bufio.Writer
	data   bool
}

func (c *SMTPConnection) writeStringWithLog(str string) {
	_, err := c.writer.WriteString(str + crlf)
	if err != nil {
		log.Printf("WriteString error: %#v", err)
	}
	log.Println(strings.ReplaceAll(str, crlf, "\\r\\n"))
	//log.Println(str + crlf)
}

func (c *SMTPConnection) handle(conn net.Conn) {
	defer conn.Close()

	c.reader = bufio.NewReader(conn)
	c.writer = bufio.NewWriter(conn)

	c.writeStringWithLog(fmt.Sprintf("220 %s ESMTP Server (Go)", hostname))
	c.writer.Flush()
	c.data = false

	for {
		line, err := c.reader.ReadString('\n')
		if err != nil {
			log.Println("already server port listen!")
			return
		}

		log.Printf("<=== %s", line)
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
				c.writeStringWithLog(fmt.Sprintf("250-%s\r\n250-PIPELINING\r\n250-SIZE 10240000\r\n250-STARTTLS\r\n250 8BITMIME", hostname))
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
				c.writeStringWithLog("250 2.0.0 Ok: queued as AAAAAAAAAA")
			case "RSET":
				c.writeStringWithLog("250 2.0.0 Ok")
			case "NOOP":
				c.writeStringWithLog("250 2.0.0 Ok")
			case "VRFY":
				c.writeStringWithLog("502 5.5.1 VRFY command is disabled")
			case "STARTTLS":
				c.writeStringWithLog("220 2.0.0 Ready to start TLS")
			default:
				if c.data == false {
					c.writeStringWithLog("500 Command not recognized")
				}
			}
		}

		c.writer.Flush()
	}
}

func listenServer() {
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatal("Listen error:", err)
	}
	defer listener.Close()

	log.Printf("SMTP server is listening on :%s\n", port)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("Accept error:", err)
			continue
		}
		c := &SMTPConnection{}
		go c.handle(conn)
	}
}

func waitForServerListen() {
	host := ip + ":" + port
	log.Print("Wait for port listen...")
	for {
		timeout := time.Second
		conn, err := net.DialTimeout("tcp", host, timeout)
		if err != nil {
			fmt.Print(".")
		}
		if conn != nil {
			conn.Close()
			break
		}
	}
	fmt.Print("\n")
}

func sendEmail() error {
	c, err := smtp.Dial(ip + ":" + port)
	if err != nil {
		log.Println("smtp dial error")
		return err
	}
	if err := c.Mail("alice@example.test"); err != nil {
		log.Println("smtp mail error")
		return err
	}
	if err := c.Rcpt("bob@example.local"); err != nil {
		log.Println("smtp rcpt error")
		return err
	}
	wc, err := c.Data()
	if err != nil {
		log.Println("smtp data error")
		return err
	}
	_, err = fmt.Fprintf(wc, "This is the email body")
	if err != nil {
		log.Println("smtp data print error")
		return err
	}
	if err = wc.Close(); err != nil {
		log.Println("smtp close print error")
		return err
	}
	if err = c.Quit(); err != nil {
		log.Println("smtp quit error")
		return err
	}
	return nil
}

func TestMain(m *testing.M) {
	go listenServer()
	waitForServerListen()
	code := m.Run()
	os.Exit(code)
}

func TestIntegration(t *testing.T) {
	err := sendEmail()
	if err != nil {
		t.Errorf("raised error: %s", err)
	}
}
