package main

import (
	"flag"
	"fmt"
	"os"
	"regexp"
	"strconv"
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
	oip     = flag.String("oip", "", "outbound ip")
	opr     = flag.String("opr", "", "outbound port range: 12000-12500")
	verFlag = flag.Bool("version", false, "show build version")
	oprRe   = regexp.MustCompile(`^([1-9][0-9]{0,5})-([1-9][0-9]{0,5})$`)
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

	trimedOpr := strings.TrimSpace(*opr)
	if trimedOpr != "" {
		matched := oprRe.FindStringSubmatch(trimedOpr)
		if len(matched) == 3 {
			s, _ := strconv.Atoi(matched[1])
			e, _ := strconv.Atoi(matched[2])
			w.OutboundPorts = &warp.PortRange{
				Start: s,
				End:   e,
			}
		} else {
			fmt.Fprintf(os.Stderr, "outbound-port-range option format is <number>-<number>\n")
			return
		}
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
