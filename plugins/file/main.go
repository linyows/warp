package main

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/linyows/warp"
)

const (
	fileCommJson string = `{"type":"comm","occurred_at":"%s","connection_id":"%s","direction":"%s","data":"%s"}
`
	fileConnJson string = `{"type":"conn","occurred_at":"%s","connection_id":"%s","from":"%s","to":"%s","elapse":"%s"}
`
)

type File struct {
	file io.Writer
}

func (f *File) Name() string {
	return "file"
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
		return nil, fmt.Errorf("os.OpenFile error: %s", err)
	}

	return f.file, nil
}

func (f *File) AfterInit() {
}

func (f *File) AfterComm(d *warp.AfterCommData) {
	writer, err := f.writer()
	if err != nil {
		fmt.Printf("[%s] %s\n", f.Name(), err)
		return
	}

	if _, err := fmt.Fprintf(writer, fileCommJson, d.OccurredAt.Format(time.RFC3339), d.ConnID, d.Direction, d.Data); err != nil {
		fmt.Printf("[%s] file append error: %s\n", f.Name(), err)
	}
}

func (f *File) AfterConn(d *warp.AfterConnData) {
	writer, err := f.writer()
	if err != nil {
		fmt.Printf("[%s] %s\n", f.Name(), err)
		return
	}

	if _, err := fmt.Fprintf(writer, fileConnJson, d.OccurredAt.Format(time.RFC3339), d.ConnID, d.MailFrom, d.MailTo, d.Elapse); err != nil {
		fmt.Printf("[%s] file append error: %s\n", f.Name(), err)
	}
}

var Hook File //nolint
