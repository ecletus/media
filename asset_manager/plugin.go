package asset_manager

import (
	"github.com/moisespsena-go/aorm"
	"github.com/moisespsena-go/pluggable"
)

type Plugin struct {
	pluggable.EventDispatcher
	DBName string
}

func (p *Plugin) OnRegister(options *pluggable.Options) {
	p.On("setup_db:gorm:"+p.DBName, func(e pluggable.PluginEventInterface) error {
		return e.Data().(*aorm.DB).AutoMigrate(&AssetManager{}).Error
	})
}
