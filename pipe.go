package warp

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"net/textproto"
	"strings"
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
	var once sync.Once

	// src ===> dst
	go func() {
		w := io.MultiWriter(p.Dst, p.Req)
		io.Copy(w, p.Src)
		once.Do(p.close())
	}()

	// src <=== dst
	go func() {
		w := io.MultiWriter(p.Src, p.Res)
		io.Copy(w, p.Dst)
		once.Do(p.close())
	}()
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

		req, err := p.data(p.Req)
		if err != nil {
			fmt.Printf("%#v\n", err)
		}
		fmt.Printf("%s\n", strings.Join(req, "\n"))
		res, err := p.data(p.Res)
		if err != nil {
			fmt.Printf("%#v\n", err)
		}
		fmt.Printf("%s\n", strings.Join(res, "\n"))
	}
}
