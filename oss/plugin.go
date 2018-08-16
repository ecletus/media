package oss

import (
	"path/filepath"

	"github.com/aghape/plug"
	"github.com/aghape/aghape"
)

type Plugin struct {
	SetupConfigKey string
}

func (p *Plugin) RequireOptions() []string {
	return []string{p.SetupConfigKey}
}

func (p *Plugin) Init(options *plug.Options) error {
	if FileSystemStorage.Base == "./data" {
		config := options.GetInterface(p.SetupConfigKey).(*qor.SetupConfig)
		FileSystemStorage.Base = filepath.Join(config.Root(), "data")
	}
	return nil
}
