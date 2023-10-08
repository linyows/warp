package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/linyows/warp"
)

var (
	version = "dev"
	commit  = ""
	date    = ""
	builtBy = ""
	ip      = flag.String("ip", "127.0.0.1", "listen ip")
	port    = flag.Int("port", 0, "listen port")
	oip     = flag.String("outbound-ip", "0.0.0.0", "outbound ip")
	storage = flag.String("storage", "", "sspecify extended storage from: mysql, sqlite, file")
	verFlag = flag.Bool("version", false, "show build version")
)

func init() {
	flag.Parse()
}

func main() {
	if *verFlag {
		fmt.Fprintf(os.Stderr, buildVersion(version, commit, date, builtBy)+"\n")
		return
	}

	w := &warp.Server{
		Addr:         *ip,
		Port:         *port,
		OutboundAddr: *oip,
	}

	switch *storage {
	case "mysql":
		w.Hooks = []warp.Hook{&warp.HookMysql{}}
	case "sqlite":
		w.Hooks = []warp.Hook{&warp.HookSqlite{}}
	case "file":
		w.Hooks = []warp.Hook{&warp.HookFile{}}
	}

	err := w.Start()
	if err != nil {
		panic(err)
	}
}

func buildVersion(version, commit, date, builtBy string) string {
	var result = version
	if commit != "" {
		result = fmt.Sprintf("%s\ncommit: %s", result, commit)
	}
	if date != "" {
		result = fmt.Sprintf("%s\nbuilt at: %s", result, date)
	}
	if builtBy != "" {
		result = fmt.Sprintf("%s\nbuilt by: %s", result, builtBy)
	}
	return result
}
