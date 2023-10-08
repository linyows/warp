package warp

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

func TestHookFileConst(t *testing.T) {
	var expect string
	var got string

	replace := func(str string) string {
		return strings.ReplaceAll(
			strings.ReplaceAll(str, "\n", ""),
			"\t", "") + "\n"
	}

	expect = replace(`
	{
		"type":"comm",
		"occurred_at":"%s",
		"connection_id":"%s",
		"direction":"%s",
		"data":"%s"
	}
	`)
	got = fileCommJson
	if got != expect {
		t.Errorf("expected %s, got %s", expect, got)
	}

	expect = replace(`
	{
		"type":"conn",
		"occurred_at":"%s",
		"connection_id":"%s",
		"from":"%s",
		"to":"%s",
		"elapse":"%s"
	}
	`)
	got = fileConnJson
	if got != expect {
		t.Errorf("expected %s, got %s", expect, got)
	}
}

func TestHookFilePrefix(t *testing.T) {
	f := &HookFile{}
	expect := "file"
	got := f.prefix()
	if got != expect {
		t.Errorf("expected %s, got %s", expect, got)
	}
}

func TestHookFileWriter(t *testing.T) {
	var tests = []struct {
		expectFileName string
		expectError    string
		envName        string
		envVal         string
	}{
		{
			expectFileName: "",
			expectError:    "missing path for file, please set `FILE_PATH`",
			envName:        "",
			envVal:         "",
		},
		{
			expectFileName: "/tmp/warp-file",
			expectError:    "",
			envName:        "FILE_PATH",
			envVal:         "/tmp/warp-file",
		},
	}

	for _, v := range tests {
		if v.envName != "" && v.envVal != "" {
			os.Setenv(v.envName, v.envVal)
			defer os.Unsetenv(v.envName)
		}

		f := &HookFile{}
		w, err := f.writer()

		if w != nil || v.expectFileName != "" {
			osf := w.(*os.File)
			if osf.Name() != v.expectFileName {
				t.Errorf("expected %s, got %s", v.expectFileName, osf.Name())
			}
		}
		if (err != nil || v.expectError != "") && fmt.Sprintf("%s", err) != v.expectError {
			t.Errorf("expected %s, got %s", v.expectError, err)
		}
	}
}

func TestHookFileAfterComm(t *testing.T) {
	ti := time.Date(2023, time.August, 16, 14, 48, 0, 0, time.UTC)
	buffer := new(bytes.Buffer)
	f := &HookFile{
		file: buffer,
	}
	data := &AfterCommData{
		ConnID:     "abcdefg",
		OccurredAt: ti,
		Data:       []byte("hello"),
		Direction:  "--",
	}
	expect := []byte(`{"type":"comm","occurred_at":"2023-08-16T14:48:00Z","connection_id":"abcdefg","direction":"--","data":"hello"}
`)
	f.AfterComm(data)
	got := buffer.Bytes()
	if !bytes.Equal(expect, got) {
		t.Errorf("expected %s, got %s", expect, got)
	}
}

func TestHookFileAfterConn(t *testing.T) {
	ti := time.Date(2023, time.August, 16, 14, 48, 0, 0, time.UTC)
	buffer := new(bytes.Buffer)
	f := &HookFile{
		file: buffer,
	}
	data := &AfterConnData{
		ConnID:     "abcdefg",
		OccurredAt: ti,
		MailFrom:   []byte("alice@example.local"),
		MailTo:     []byte("bob@example.test"),
		Elapse:     20,
	}
	expect := []byte(`{"type":"conn","occurred_at":"2023-08-16T14:48:00Z","connection_id":"abcdefg","from":"alice@example.local","to":"bob@example.test","elapse":"20 msec"}
`)
	f.AfterConn(data)
	got := buffer.Bytes()
	if !bytes.Equal(expect, got) {
		t.Errorf("expected %s, got %s", expect, got)
	}
}
