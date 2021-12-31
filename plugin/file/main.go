package main

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/linyows/warp"
)

const (
	commJson string = "{\"type\":\"comm\",\"occurred_at\":\"%s\",\"connection_id\":\"%s\",\"direction\":\"%s\",\"data\":\"%s\"}\n"
	connJson string = "{\"type\":\"conn\"\"occurred_at\":\"%s\",\"connection_id\":\"%s\",\"from\":\"%s\",\"to\":\"%s\"\n}"
)

type File struct {
	file *os.File
}

func (f *File) writer() io.Writer {
	if f.file != nil {
		return f.file
	}

	path := os.Getenv("FILE_PATH")
	if len(path) == 0 {
		panic("missing path for file")
	}

	var err error
	f.file, err = os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}

	return f.file
}

func (f *File) AfterComm(d *warp.AfterCommData) {
	if _, err := fmt.Fprintf(f.writer(), commJson, d.OccurredAt.Format(time.RFC3339), d.ConnID, d.Direction, d.Data); err != nil {
		fmt.Printf("file append error: %#v\n", err)
	}
}

func (f *File) AfterConn(d *warp.AfterConnData) {
	if _, err := fmt.Fprintf(f.writer(), connJson, d.OccurredAt.Format(time.RFC3339), d.ConnID, d.MailFrom, d.MailTo); err != nil {
		fmt.Printf("file append error: %#v\n", err)
	}
}

var Hook File
