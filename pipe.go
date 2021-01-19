package warp

import (
	"io"
	"log"
	"net"
	"sync"
)

type Pipe struct {
	Src *net.TCPConn
	Dst *net.TCPConn
	//ok  bool
}

func (p *Pipe) Do() {
	log.Print("start proxy")
	//p.ok = true

	close := func() {
		defer p.Dst.Close()
		defer p.Src.Close()
		defer log.Print("connection closed")
		//p.ok = false
	}
	var once sync.Once

	go func() {
		io.Copy(p.Dst, p.Src)
		once.Do(close)
	}()

	go func() {
		io.Copy(p.Src, p.Dst)
		once.Do(close)
	}()

	/*
		go func() {
			var rwc io.ReadWriteCloser = p.Dst
			tp := textproto.NewConn(rwc)
			for p.ok {
				line, err := tp.ReadLine()
				if err != nil {
					if err == io.EOF {
						log.Print("EOF")
						continue
					}
					if neterr, ok := err.(net.Error); ok && neterr.Timeout() {
						log.Printf("%#v", err)
						continue
					}
					continue
				}
				fmt.Printf("%s", line)
			}
		}()
	*/

	log.Print("end proxy")
}
