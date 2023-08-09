package integration

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	go SMTPServer()
	WaitForServerListen()
	code := m.Run()
	os.Exit(code)
}

func TestIntegration(t *testing.T) {
	err := SendEmail()
	if err != nil {
		t.Errorf("raised error: %s", err)
	}
}
