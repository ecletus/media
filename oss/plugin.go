package oss

import (
	"path/filepath"

	"github.com/ecletus/db"

	"github.com/ecletus/core"
	"github.com/ecletus/plug"
)

type Plugin struct {
	db.DBNames
	plug.EventDispatcher
	SetupConfigKey string
}

func (p *Plugin) RequireOptions() []string {
	return []string{p.SetupConfigKey}
}

func (p *Plugin) Init(options *plug.Options) error {
	if FileSystemStorage.Base == "./data" {
		config := options.GetInterface(p.SetupConfigKey).(*core.SetupConfig)
		FileSystemStorage.Base = filepath.Join(config.Root(), "data")
	}
	return nil
}

func (p *Plugin) OnRegister() {
	db.Events(p).DBOnInitGorm(func(e *db.DBEvent) {
		RegisterCallbacks(e.DB.DB)
	})
}
