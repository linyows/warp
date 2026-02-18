package warp

import (
	"bytes"
	"fmt"
	"log"
	"strings"
	"testing"
	"time"
)

func TestIntegration(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}

	ip := "127.0.0.1"
	warpPort := 10025
	smtpPort := 11025
	hostname := "example.local"

	var (
		warpLog bytes.Buffer
		smtpLog bytes.Buffer
	)

	messages := make(chan ReceivedMessage, 1)

	go func() {
		specifiedDstIP = ip
		specifiedDstPort = smtpPort
		w := &Server{
			Addr:    ip,
			Port:    warpPort,
			Verbose: true,
			log:     log.New(&warpLog, "", log.Ldate|log.Ltime|log.Lmicroseconds),
		}
		if err := w.Start(); err != nil {
			t.Errorf("warp raised error: %s", err)
		}
	}()

	go func() {
		s := &SMTPServer{
			IP:       ip,
			Port:     smtpPort,
			Hostname: hostname,
			log:      log.New(&smtpLog, "", log.Ldate|log.Ltime|log.Lmicroseconds),
			OnMessage: func(msg ReceivedMessage) {
				messages <- msg
			},
		}
		if err := s.Serve(); err != nil {
			t.Errorf("smtp server raised error: %s", err)
		}
	}()

	WaitForServerListen(ip, warpPort)
	WaitForServerListen(ip, smtpPort)

	c := &SMTPClient{IP: ip, Port: warpPort}
	err := c.SendEmail()

	fmt.Printf("\nWarp Server:\n%s", &warpLog)
	fmt.Printf("\nSMTP Server:\n%s\n", &smtpLog)

	if err != nil {
		t.Fatalf("smtp client raised error: %s", err)
	}

	select {
	case msg := <-messages:
		if msg.MailFrom != "alice@example.test" {
			t.Errorf("MailFrom = %q, want %q", msg.MailFrom, "alice@example.test")
		}
		if !strings.Contains(string(msg.Data), "Subject: Test") {
			t.Errorf("Data does not contain 'Subject: Test': %s", string(msg.Data[:min(len(msg.Data), 200)]))
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for message from SMTP server")
	}
}
