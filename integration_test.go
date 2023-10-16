package warp

import (
	"bytes"
	"fmt"
	"log"
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
		}
		if err := s.Serve(); err != nil {
			t.Errorf("smtp server raised error: %s", err)
		}
	}()

	WaitForServerListen(ip, warpPort)
	WaitForServerListen(ip, smtpPort)

	c := &SMTPClient{IP: ip, Port: warpPort}
	err := c.SendEmail()
	time.Sleep(1 * time.Second)

	fmt.Printf("\nWarp Server:\n%s", &warpLog)
	fmt.Printf("\nSMTP Server:\n%s\n", &smtpLog)

	if err != nil {
		t.Errorf("smtp client raised error: %s", err)
	}
}
