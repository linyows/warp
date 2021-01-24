package warp

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"net/textproto"
	"sync"
)

type Pipe struct {
	Src *net.TCPConn
	Dst *net.TCPConn
}

func (p *Pipe) Do() {
	var once sync.Once

	// src ===> dst
	go func() {
		p.Copy(p.Dst, p.Src, false)
		once.Do(p.close())
	}()

	// src <=== dst
	go func() {
		p.Copy(p.Src, p.Dst, true)
		once.Do(p.close())
	}()
}

func (p *Pipe) Copy(dst io.Writer, src io.Reader, backward bool) (written int64, err error) {
	size := 32 * 1024
	if l, ok := src.(*io.LimitedReader); ok && int64(size) > l.N {
		if l.N < 1 {
			size = 1
		} else {
			size = int(l.N)
		}
	}
	buf := make([]byte, size)

	for {
		nr, er := src.Read(buf)
		if nr > 0 {
			if backward {
				if bytes.Contains(buf, []byte("STARTTLS")) {
					old := []byte("250-STARTTLS\r\n")
					buf = bytes.Replace(buf, old, []byte(""), 1)
					nr = nr - len(old)
				}
				fmt.Printf("<===\n%s\n", buf)
			} else {
				fmt.Printf("===>\n%s\n", buf)
			}
			nw, ew := dst.Write(buf[0:nr])
			if nw > 0 {
				written += int64(nw)
			}
			if ew != nil {
				err = ew
				break
			}
			if nr != nw {
				err = io.ErrShortWrite
				break
			}
		}
		if er != nil {
			if er != io.EOF {
				err = er
			}
			break
		}
	}

	return written, err
}

func (p *Pipe) data(b *bytes.Buffer) ([]string, error) {
	var data []string
	r := textproto.NewReader(bufio.NewReader(b))
	for {
		line, err := r.ReadLine()
		if err != nil {
			if err == io.EOF {
				break
			}
			return data, err
		}
		data = append(data, line)
	}
	return data, nil
}

func (p *Pipe) close() func() {
	return func() {
		defer p.Dst.Close()
		defer p.Src.Close()
		defer log.Print("connection closed")
	}
}
