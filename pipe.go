package warp

import (
	"bytes"
	"io"
	"log"
	"net"
	"sync"
)

type Pipe struct {
	Src *net.TCPConn
	Dst *net.TCPConn
	Req *bytes.Buffer
	Res *bytes.Buffer
}

func (p *Pipe) Do() {
	p.Req = new(bytes.Buffer)
	p.Res = new(bytes.Buffer)

	close := func() {
		defer p.Dst.Close()
		defer p.Src.Close()
		defer log.Print("connection closed")
	}
	var once sync.Once

	// src ===> dst
	go func() {
		w := io.MultiWriter(p.Dst, p.Req)
		io.Copy(w, p.Src)
		once.Do(close)
	}()

	// src <=== dst
	go func() {
		w := io.MultiWriter(p.Src, p.Res)
		io.Copy(w, p.Dst)
		once.Do(close)
	}()
}
