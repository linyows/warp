package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof" // registers /debug/pprof/* on http.DefaultServeMux
	"os"
	"strings"
	"time"

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
	pprofAddr = flag.String("pprof", "", "expose net/http/pprof on host:port (e.g. 127.0.0.1:6060); empty = disabled. Bind to a loopback or otherwise restricted address — pprof endpoints leak heap/goroutine state and are not access-controlled.")
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

	if *pprofAddr != "" {
		startPprofServer(*pprofAddr)
	}

	w := &warp.Server{
		Addr:             *ip,
		Port:             *port,
		OutboundAddr:     *oip,
		Verbose:          *verbose,
		MessageSizeLimit: *maxSize,
	}

	if *plugins != "" {
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

// startPprofServer launches the net/http/pprof endpoints on addr in a
// background goroutine. Errors from ListenAndServe are logged but do not
// abort the warp process — pprof is a diagnostic tool, not a hard
// dependency of the SMTP proxy.
func startPprofServer(addr string) {
	// Conservative timeouts so a stuck pprof client cannot exhaust file
	// descriptors. The handler itself can stream long captures
	// (CPU profile with ?seconds=30 etc.) so write timeout is set
	// generously enough to allow them, while idle/read are kept short.
	srv := &http.Server{
		Addr:              addr,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      5 * time.Minute,
		IdleTimeout:       30 * time.Second,
	}
	go func() {
		log.Printf("pprof endpoints listening on http://%s/debug/pprof/", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("pprof server error: %s", err)
		}
	}()
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
