package oss

import (
	"github.com/ecletus/admin"
	"github.com/ecletus/core"
	"github.com/ecletus/core/resource"
	"github.com/ecletus/media"
	"github.com/moisespsena-go/aorm"
)

func init() {
	aorm.StructFieldMethodCallbacks.RegisterFieldType(&Doc{})
}

type Doc struct {
	OSS
}

func (*Doc) FileExts() []string {
	return []string{"pdf", "doc", "docx", "odt"}
}

func (d *Doc) ContextScan(ctx *core.Context, data interface{}) (err error) {
	return d.Set(media.NewContext(d), data)
}

func (d *Doc) ConfigureQorMetaBeforeInitialize(metaor resource.Metaor) {
	if meta, ok := metaor.(*admin.Meta); ok {
		d.OSS.ConfigureQorMetaBeforeInitialize(meta)
		meta.DefaultValueFunc = func(recorde interface{}, context *core.Context) interface{} {
			return Doc{}
		}
	}
}
