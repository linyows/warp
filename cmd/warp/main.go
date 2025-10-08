package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

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
	plugins = flag.String("plugins", "", "use plugin names: mysql, sqlite, file, slack")
	maxSize = flag.Int("message-size-limit", 10240000, "The maximal size in bytes of a message")
	verbose = flag.Bool("verbose", false, "verbose logging")
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
		Addr:             *ip,
		Port:             *port,
		OutboundAddr:     *oip,
		Verbose:          *verbose,
		MessageSizeLimit: *maxSize,
	}

	if "" != *plugins {
		pp := strings.Split(*plugins, ",")
		for i := range pp {
			pp[i] = strings.TrimSpace(pp[i])
		}
		w.Plugins = pp
	}

	err := w.Start()
	if err != nil {
		panic(err)
	}
}

func buildVersion(version, commit, date, builtBy string) string {
	result := version
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
