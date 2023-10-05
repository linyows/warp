package main

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/linyows/warp"
)

const (
	prefix   string = "file-plugin"
	commJson string = `{"type":"comm","occurred_at":"%s","connection_id":"%s","direction":"%s","data":"%s"}
`
	connJson string = `{"type":"conn","occurred_at":"%s","connection_id":"%s","from":"%s","to":"%s","elapse":"%s"}
`
)

type File struct {
	file io.Writer
}

func (f *File) writer() (io.Writer, error) {
	if f.file != nil {
		return f.file, nil
	}

	path := os.Getenv("FILE_PATH")
	if len(path) == 0 {
		return nil, fmt.Errorf("missing path for file, please set `FILE_PATH`")
	}

	var err error
	f.file, err = os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("os.OpenFile error: %#v\n", err)
	}

	return f.file, nil
}

func (f *File) AfterInit() {
}

func (f *File) AfterComm(d *warp.AfterCommData) {
	writer, err := f.writer()
	if err != nil {
		fmt.Printf("[%s] %#v\n", prefix, err)
		return
	}

	if _, err := fmt.Fprintf(writer, commJson, d.OccurredAt.Format(time.RFC3339), d.ConnID, d.Direction, d.Data); err != nil {
		fmt.Printf("[%s] file append error: %#v\n", prefix, err)
	}
}

func (f *File) AfterConn(d *warp.AfterConnData) {
	writer, err := f.writer()
	if err != nil {
		fmt.Printf("[%s] %#v\n", prefix, err)
		return
	}

	if _, err := fmt.Fprintf(writer, connJson, d.OccurredAt.Format(time.RFC3339), d.ConnID, d.MailFrom, d.MailTo, d.Elapse); err != nil {
		fmt.Printf("[%s] file append error: %#v\n", prefix, err)
	}
}

var Hook File //nolint
