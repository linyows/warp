package warp

import (
	"testing"
)

func TestIntegration(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}

	ip := "127.0.0.1"
	warpPort := 10025
	smtpPort := 11025
	hostname := "example.local"

	go func() {
		specifiedDstIP = ip
		specifiedDstPort = smtpPort
		w := &Server{Addr: ip, Port: warpPort}
		if err := w.Start(); err != nil {
			t.Errorf("warp raised error: %s", err)
		}
	}()

	go func() {
		s := &SMTPServer{IP: ip, Port: smtpPort, Hostname: hostname}
		if err := s.Serve(); err != nil {
			t.Errorf("smtp server raised error: %s", err)
		}
	}()

	WaitForServerListen(ip, warpPort)
	WaitForServerListen(ip, smtpPort)

	c := &SMTPClient{IP: ip, Port: warpPort}
	if err := c.SendEmail(); err != nil {
		t.Errorf("smtp client raised error: %s", err)
	}
}
