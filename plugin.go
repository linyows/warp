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
	pluginDirName string = "plugin"
	pluginVarName string = "Hook"
	TimeFormat    string = "2006-01-02T15:04:05.999999"
)

type Hook interface {
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

func pluginDirExists() bool {
	_, err := os.Stat(pluginDirName)
	return err == nil
}

func loadPlugin(name string) (Hook, error) {
	p := path.Join(pluginDirName, name)
	plug, err := plugin.Open(p)
	if err != nil {
		return nil, err
	}

	symbol, err := plug.Lookup(pluginVarName)
	if err != nil {
		return nil, err
	}

	log.Printf("plugin loaded: %s", p)
	return symbol.(Hook), nil
}

func loadPlugins() ([]Hook, error) {
	var plugins []Hook

	if !pluginDirExists() {
		return plugins, nil
	}

	files, err := ioutil.ReadDir(pluginDirName)
	if err != nil {
		return plugins, err
	}

	for _, f := range files {
		if !f.Mode().IsRegular() {
			continue
		}
		n := f.Name()
		if filepath.Ext(n) != ".so" {
			continue
		}

		plug, err := loadPlugin(n)
		if err != nil {
			fmt.Printf("plugin load error(%s): %#v\n", n, err)
			continue
		}

		plugins = append(plugins, plug)
	}

	return plugins, nil
}
