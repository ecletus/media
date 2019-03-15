package media_library

import (
	"github.com/ecletus/db"
	"github.com/ecletus/plug"
)

type Plugin struct {
	db.DBNames
	plug.EventDispatcher
}

func (p *Plugin) OnRegister() {
	db.Events(p).DBOnMigrate(func(e *db.DBEvent) error {
		return e.AutoMigrate(&QorMediaLibrary{}).Error
	})
}
