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

func sendEmail() error {
	c, err := smtp.Dial("localhost:2525")
	if err != nil {
		return err
	}
	if err := c.Mail("sender@example.org"); err != nil {
		return err
	}
	if err := c.Rcpt("recipient@example.net"); err != nil {
		return err
	}
	wc, err := c.Data()
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(wc, "This is the email body")
	if err != nil {
		return err
	}
	if err = wc.Close(); err != nil {
		return err
	}
	if err = c.Quit(); err != nil {
		return err
	}
	return nil
}

func handleClient(conn net.Conn) {
	defer conn.Close()

	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)
	hostname := "recipient"

	str := fmt.Sprintf("220 %s ESMTP Server (Go)\r\n", hostname)
	writer.WriteString(str)
	log.Print(str)
	writer.Flush()
	data := false

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Read error:", err)
			return
		}

		line = strings.TrimSpace(line)
		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}

		commands := strings.Split(line, "\r\n")
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
			log.Printf("<=== %s\n", v)

			switch first {
			case "HELO":
				str := fmt.Sprintf("250 Hello %s\r\n", parts[1])
				writer.WriteString(str)
				log.Print(str)
			case "EHLO":
				str := fmt.Sprintf(`250-%s
250-PIPELINING
250-SIZE 10240000
250-VRFY
250-ETRN
250-STARTTLS
250-ENHANCEDSTATUSCODES
250-8BITMIME
250-DSN
250-SMTPUTF8
250 CHUNKING
`, hostname)
				str = strings.ReplaceAll(str, "\n", "\r\n")
				writer.WriteString(str)
				log.Print(strings.ReplaceAll(str, "\r\n", "\\r\\n"))
			case "MAIL":
				if strings.Contains(second, "FROM:") {
					str := "250 2.1.0 Ok\r\n"
					writer.WriteString(str)
					log.Print(strings.ReplaceAll(str, "\r\n", "\\r\\n"))
				}
			case "RCPT":
				str := "250 2.1.5 Ok\r\n"
				if strings.Contains(second, "TO:") {
					writer.WriteString(str)
					log.Print(strings.ReplaceAll(str, "\r\n", "\\r\\n"))
				}
			case "DATA":
				data = true
				str := "354 End data with <CR><LF>.<CR><LF>\r\n"
				writer.WriteString(str)
				log.Print(strings.ReplaceAll(str, "\r\n", "\\r\\n"))
			case "QUIT":
				str := "221 2.0.0 Bye\r\n"
				writer.WriteString(str)
				log.Print(str)
				writer.Flush()
				return
			case ".":
				data = false
				str := "250 2.0.0 Ok: queued as 76DAD4113D\r\n"
				writer.WriteString(str)
				log.Print(str)
			case "RSET":
				str := "250 2.0.0 Ok\r\n"
				writer.WriteString(str)
				log.Print(str)
			case "NOOP":
				str := "250 2.0.0 Ok\r\n"
				writer.WriteString(str)
				log.Print(str)
			case "VRFY":
				str := "502 5.5.1 VRFY command is disabled\r\n"
				writer.WriteString(str)
				log.Print(str)
			case "STARTTLS":
				str := "220 2.0.0 Ready to start TLS\r\n"
				writer.WriteString(str)
				log.Print(str)
			default:
				if data == false {
					str := "500 Command not recognized\r\n"
					writer.WriteString(str)
					log.Print(str)
				}
			}
		}

		writer.Flush()
	}
}

func listenServer() {
	listener, err := net.Listen("tcp", ":2525")
	if err != nil {
		log.Fatal("Listen error:", err)
	}
	defer listener.Close()

	log.Print("SMTP server is listening on :2525\r\n")

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("Accept error:", err)
			continue
		}
		go handleClient(conn)
	}
}

func waitForServerListen() {
	host := "localhost:2525"
	fmt.Printf("wait server listen(%s)...", host)
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

func TestMain(m *testing.M) {
	go listenServer()
	//waitForServerListen()
	code := m.Run()
	os.Exit(code)
}

func TestRequest(t *testing.T) {
	err := sendEmail()
	if err != nil {
		t.Errorf("raised error: %s", err)
	}
}
