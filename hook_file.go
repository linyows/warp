package warp

import (
	"fmt"
	"io"
	"os"
	"time"
)

const (
	fileCommJson string = `{"type":"comm","occurred_at":"%s","connection_id":"%s","direction":"%s","data":"%s"}
`
	fileConnJson string = `{"type":"conn","occurred_at":"%s","connection_id":"%s","from":"%s","to":"%s","elapse":"%s"}
`
)

type HookFile struct {
	file io.Writer
}

func (h *HookFile) Name() string {
	return "file"
}

func (h *HookFile) writer() (io.Writer, error) {
	if h.file != nil {
		return h.file, nil
	}

	path := os.Getenv("FILE_PATH")
	if len(path) == 0 {
		return nil, fmt.Errorf("missing path for file, please set `FILE_PATH`")
	}

	var err error
	h.file, err = os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("os.OpenFile error: %s\n", err)
	}

	return h.file, nil
}

func (h *HookFile) AfterInit() {
}

func (h *HookFile) AfterComm(d *AfterCommData) {
	writer, err := h.writer()
	if err != nil {
		fmt.Printf("[%s] %s\n", h.Name(), err)
		return
	}

	if _, err := fmt.Fprintf(writer, fileCommJson, d.OccurredAt.Format(time.RFC3339), d.ConnID, d.Direction, d.Data); err != nil {
		fmt.Printf("[%s] file append error: %s\n", h.Name(), err)
	}
}

func (h *HookFile) AfterConn(d *AfterConnData) {
	writer, err := h.writer()
	if err != nil {
		fmt.Printf("[%s] %s\n", h.Name(), err)
		return
	}

	if _, err := fmt.Fprintf(writer, fileConnJson, d.OccurredAt.Format(time.RFC3339), d.ConnID, d.MailFrom, d.MailTo, d.Elapse); err != nil {
		fmt.Printf("[%s] file append error: %s\n", h.Name(), err)
	}
}
