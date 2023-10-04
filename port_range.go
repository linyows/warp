package warp

import (
	"fmt"
	"math/rand"
	"net"
)

type PortRange struct {
	Start int
	End   int
}

func (p *PortRange) TakeOut() (int, error) {
	diff := p.End - p.Start
	for i := p.Start; i <= p.End; i++ {
		// Incremental port checks is port duplicate, when consecutive send. so using random port checks.
		port := p.Start + rand.Intn(diff)
		if ok := isPortAvailable(port); ok {
			return i, nil
		}
	}
	return 0, fmt.Errorf("not found open port by %d-%d", p.Start, p.End)
}

func isPortAvailable(port int) bool {
	_, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	return err == nil
}
