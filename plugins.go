package warp

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"plugin"
	"time"
)

const (
	pluginVarName string = "Hook"
	TimeFormat    string = "2006-01-02T15:04:05.999999"
)

type Hook interface {
	AfterInit()
	AfterComm(*AfterCommData)
	AfterConn(*AfterConnData)
}

type AfterCommData struct {
	ConnID     string
	OccurredAt time.Time
	Data
	Direction
}

type AfterConnData struct {
	ConnID     string
	OccurredAt time.Time
	MailFrom   []byte
	MailTo     []byte
	Elapse
}

type Plugins struct {
	path  string
	hooks []Hook
}

func (p *Plugins) isDirExists() bool {
	_, err := os.Stat(p.path)
	return err == nil
}

func (p *Plugins) setPath() {
	p.path = "/opt/warp/plugins"
	path := os.Getenv("PLUGIN_PATH")
	if path != "" {
		p.path = path
	}
}

func (p *Plugins) lookup(name string) (Hook, error) {
	pp := path.Join(p.path, name)
	plug, err := plugin.Open(pp)
	if err != nil {
		return nil, err
	}

	symbol, err := plug.Lookup(pluginVarName)
	if err != nil {
		return nil, err
	}

	log.Printf("plugin loaded: %s", pp)
	return symbol.(Hook), nil
}

func (p *Plugins) load() error {
	p.setPath()

	if !p.isDirExists() {
		return nil
	}

	files, err := ioutil.ReadDir(p.path)
	if err != nil {
		return err
	}

	for _, f := range files {
		if !f.Mode().IsRegular() {
			continue
		}
		n := f.Name()
		if filepath.Ext(n) != ".so" {
			continue
		}

		plug, err := p.lookup(n)
		if err != nil {
			fmt.Printf("plugin load error(%s): %s\n", n, err)
			continue
		}

		plug.AfterInit()
		p.hooks = append(p.hooks, plug)
	}

	return nil
}
