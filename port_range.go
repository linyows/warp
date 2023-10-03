package warp

import (
	"fmt"
	"net"
	"strconv"
	"time"
)

type PortRange struct {
	start int
	end   int
}

func (p *PortRange) TakeOut(host string) (int, error) {
	timeout := time.Second

	for i := p.start; i <= p.end; i++ {
		address := net.JoinHostPort(host, strconv.Itoa(i))
		if ok := isAvailablePort(address, timeout); ok {
			return i, nil
		}
	}

	return 0, fmt.Errorf("not found open port by %d-%d", p.start, p.end)
}

func isAvailablePort(address string, timeout time.Duration) bool {
	conn, err := net.DialTimeout("tcp", address, timeout)
	if conn != nil {
		defer conn.Close()
	}
	return err != nil
}
