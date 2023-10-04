package warp

import (
	"fmt"
	"net"
	"testing"
)

func TestTakeOut(t *testing.T) {
	port := 30000
	start := port - 10
	end := port + 10

	r := &PortRange{Start: start, End: end}
	_, err := r.TakeOut()

	if err != nil {
		t.Errorf("TakeOut() expects a port number return, but got error: %s", err.Error())
	}
}

func TestIsPortAvailable(t *testing.T) {
	used := 30001
	unused := 30002

	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", used))
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	if isPortAvailable(used) {
		t.Error("port was listened, but isPortAvailable returns true")
	}

	if !isPortAvailable(unused) {
		t.Error("port was not listened, but isPortAvailable returns false")
	}
}
