package warp

import (
	"bytes"
	"fmt"
	"log"
	"net/smtp"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// testHook implements Hook interface for E2E tests.
type testHook struct {
	mu        sync.Mutex
	commCalls []*AfterCommData
	connCalls []*AfterConnData
}

func (h *testHook) Name() string    { return "test" }
func (h *testHook) AfterInit()      {}

func (h *testHook) AfterComm(d *AfterCommData) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.commCalls = append(h.commCalls, d)
}

func (h *testHook) AfterConn(d *AfterConnData) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.connCalls = append(h.connCalls, d)
}

func (h *testHook) waitForConnCalls(n int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		h.mu.Lock()
		count := len(h.connCalls)
		h.mu.Unlock()
		if count >= n {
			return true
		}
		time.Sleep(50 * time.Millisecond)
	}
	return false
}

func (h *testHook) getConnCalls() []*AfterConnData {
	h.mu.Lock()
	defer h.mu.Unlock()
	result := make([]*AfterConnData, len(h.connCalls))
	copy(result, h.connCalls)
	return result
}

func (h *testHook) findConnCall(mailFrom string, timeout time.Duration) *AfterConnData {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		h.mu.Lock()
		for _, c := range h.connCalls {
			if string(c.MailFrom) == mailFrom {
				h.mu.Unlock()
				return c
			}
		}
		h.mu.Unlock()
		time.Sleep(50 * time.Millisecond)
	}
	return nil
}

func (h *testHook) getCommCalls() []*AfterCommData {
	h.mu.Lock()
	defer h.mu.Unlock()
	result := make([]*AfterCommData, len(h.commCalls))
	copy(result, h.commCalls)
	return result
}

// testEnv encapsulates the warp server, test SMTP server, and hook for E2E tests.
type testEnv struct {
	ip       string
	warpPort int
	smtpPort int
	hostname string
	hook     *testHook
	messages chan ReceivedMessage
	warpLog  bytes.Buffer
	smtpLog  bytes.Buffer
}

var nextPort int32 = 20000

func allocPorts() (int, int) {
	p := int(atomic.AddInt32(&nextPort, 2))
	return p - 1, p
}

func setupTestEnv(t *testing.T) *testEnv {
	t.Helper()

	warpPort, smtpPort := allocPorts()
	env := &testEnv{
		ip:       "127.0.0.1",
		warpPort: warpPort,
		smtpPort: smtpPort,
		hostname: "example.local",
		hook:     &testHook{},
		messages: make(chan ReceivedMessage, 10),
	}

	go func() {
		specifiedDstIP = env.ip
		specifiedDstPort = env.smtpPort
		w := &Server{
			Addr:    env.ip,
			Port:    env.warpPort,
			Verbose: true,
			Hooks:   []Hook{env.hook},
			log:     log.New(&env.warpLog, "", log.Ldate|log.Ltime|log.Lmicroseconds),
		}
		if err := w.Start(); err != nil {
			t.Errorf("warp raised error: %s", err)
		}
	}()

	go func() {
		s := &SMTPServer{
			IP:       env.ip,
			Port:     env.smtpPort,
			Hostname: env.hostname,
			log:      log.New(&env.smtpLog, "", log.Ldate|log.Ltime|log.Lmicroseconds),
			OnMessage: func(msg ReceivedMessage) {
				env.messages <- msg
			},
		}
		if err := s.Serve(); err != nil {
			t.Errorf("smtp server raised error: %s", err)
		}
	}()

	WaitForServerListen(env.ip, env.warpPort)
	WaitForServerListen(env.ip, env.smtpPort)

	return env
}

func (env *testEnv) sendEmail(t *testing.T, from, to, subject, body string) {
	t.Helper()

	s, err := smtp.Dial(fmt.Sprintf("%s:%d", env.ip, env.warpPort))
	if err != nil {
		t.Fatalf("smtp.Dial error: %v", err)
	}
	defer func() {
		if err := s.Quit(); err != nil {
			t.Logf("QUIT error: %v", err)
		}
	}()

	if err := s.Mail(from); err != nil {
		t.Fatalf("MAIL FROM error: %v", err)
	}
	if err := s.Rcpt(to); err != nil {
		t.Fatalf("RCPT TO error: %v", err)
	}
	wc, err := s.Data()
	if err != nil {
		t.Fatalf("DATA error: %v", err)
	}

	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s", from, to, subject, body)
	if _, err := wc.Write([]byte(msg)); err != nil {
		t.Fatalf("DATA write error: %v", err)
	}
	if err := wc.Close(); err != nil {
		t.Fatalf("DATA close error: %v", err)
	}
}

func (env *testEnv) waitForMessage(t *testing.T, timeout time.Duration) ReceivedMessage {
	t.Helper()
	select {
	case msg := <-env.messages:
		return msg
	case <-time.After(timeout):
		t.Fatalf("timed out waiting for message (timeout: %s)", timeout)
		return ReceivedMessage{}
	}
}

func TestE2E(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}

	env := setupTestEnv(t)

	t.Run("MessageRelayIntegrity", func(t *testing.T) {
		from := "sender@example.test"
		to := "receiver@example.local"
		subject := "Integrity Test"
		body := "Hello, this is a relay integrity test."

		env.sendEmail(t, from, to, subject, body)
		msg := env.waitForMessage(t, 5*time.Second)

		if msg.MailFrom != from {
			t.Errorf("MailFrom = %q, want %q", msg.MailFrom, from)
		}
		if len(msg.RcptTo) != 1 || msg.RcptTo[0] != to {
			t.Errorf("RcptTo = %v, want [%q]", msg.RcptTo, to)
		}

		data := string(msg.Data)
		if !strings.Contains(data, "Subject: "+subject) {
			t.Errorf("Data does not contain expected Subject header:\n%s", data[:min(len(data), 300)])
		}
		if !strings.Contains(data, body) {
			t.Errorf("Data does not contain expected body:\n%s", data[:min(len(data), 300)])
		}
	})

	t.Run("MetadataExtraction", func(t *testing.T) {
		// Find the conn call matching the email we sent (probe connections have "unknown")
		emailConn := env.hook.findConnCall("sender@example.test", 5*time.Second)
		if emailConn == nil {
			t.Fatal("timed out waiting for AfterConn hook call with expected MailFrom")
		}

		if string(emailConn.MailFrom) != "sender@example.test" {
			t.Errorf("AfterConn MailFrom = %q, want %q", emailConn.MailFrom, "sender@example.test")
		}
		if string(emailConn.MailTo) != "receiver@example.local" {
			t.Errorf("AfterConn MailTo = %q, want %q", emailConn.MailTo, "receiver@example.local")
		}
	})

	t.Run("HookCallbacks", func(t *testing.T) {
		comms := env.hook.getCommCalls()
		if len(comms) == 0 {
			t.Fatal("no AfterComm calls recorded")
		}

		directions := make(map[Direction]bool)
		for _, c := range comms {
			directions[c.Direction] = true
			if c.ConnID == "" {
				t.Error("AfterComm has empty ConnID")
			}
			if c.OccurredAt.IsZero() {
				t.Error("AfterComm has zero OccurredAt")
			}
		}

		for _, d := range []Direction{onPxy, srcToDst} {
			if !directions[d] {
				t.Errorf("AfterComm not called with Direction %q", d)
			}
		}
	})

	t.Run("StartTLSStripping", func(t *testing.T) {
		comms := env.hook.getCommCalls()

		var dstToPxyHasStartTLS bool
		var pxyToSrcHasStartTLS bool
		for _, c := range comms {
			dataStr := string(c.Data)
			if c.Direction == dstToPxy && strings.Contains(dataStr, "STARTTLS") {
				dstToPxyHasStartTLS = true
			}
			if c.Direction == pxyToSrc && strings.Contains(dataStr, "STARTTLS") {
				pxyToSrcHasStartTLS = true
			}
		}

		if !dstToPxyHasStartTLS {
			t.Error("expected STARTTLS in dstToPxy communication, but not found")
		}
		if pxyToSrcHasStartTLS {
			t.Error("STARTTLS should be stripped from pxyToSrc communication, but was found")
		}
	})

	t.Run("MultipleEmails", func(t *testing.T) {
		connCountBefore := len(env.hook.getConnCalls())

		env.sendEmail(t, "user2@example.test", "dest2@example.local", "Second Email", "Body of second email")
		msg2 := env.waitForMessage(t, 5*time.Second)
		if msg2.MailFrom != "user2@example.test" {
			t.Errorf("2nd email MailFrom = %q, want %q", msg2.MailFrom, "user2@example.test")
		}
		if !strings.Contains(string(msg2.Data), "Body of second email") {
			t.Error("2nd email Data does not contain expected body")
		}

		env.sendEmail(t, "user3@example.test", "dest3@example.local", "Third Email", "Body of third email")
		msg3 := env.waitForMessage(t, 5*time.Second)
		if msg3.MailFrom != "user3@example.test" {
			t.Errorf("3rd email MailFrom = %q, want %q", msg3.MailFrom, "user3@example.test")
		}
		if !strings.Contains(string(msg3.Data), "Body of third email") {
			t.Error("3rd email Data does not contain expected body")
		}

		if !env.hook.waitForConnCalls(connCountBefore+2, 5*time.Second) {
			t.Error("timed out waiting for AfterConn hook calls for multiple emails")
		}
	})
}
