package main

import (
	"flag"

	"github.com/linyows/warp"
)

func main() {
	var (
		ip   = flag.String("ip", "127.0.0.1", "listen ip")
		port = flag.Int("port", 0, "listen port")
	)
	flag.Parse()
	w := &warp.Server{Addr: *ip, Port: *port}
	err := w.Start()
	if err != nil {
		panic(err)
	}
}
