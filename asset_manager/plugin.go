package asset_manager

import (
	"github.com/jinzhu/gorm"
	"github.com/moisespsena/go-pluggable"
)

type Plugin struct {
	pluggable.EventDispatcher
	DBName string
}

func (p *Plugin) OnRegister(dis pluggable.PluginEventDispatcherInterface) {
	p.On("setup_db:gorm:" + p.DBName, func(e pluggable.PluginEventInterface) error {
		return e.Data().(*gorm.DB).AutoMigrate(&AssetManager{}).Error
	})
}