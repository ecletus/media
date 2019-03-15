package media_library

import (
	"database/sql/driver"
	"encoding/json"

	"github.com/aghape/admin"
	"github.com/aghape/core"
	"github.com/aghape/core/helpers"
	"github.com/aghape/core/resource"
	"github.com/aghape/media"
	"github.com/aghape/media/oss"
	"github.com/moisespsena-go/aorm"
)

type MediaLibraryStorageAttributes struct {
	Video        string
	SelectedType string
	Description  string
}

type MediaLibraryStorage struct {
	oss.Image
	MediaLibraryStorageAttributes
}

func (b *MediaLibraryStorage) IsZero() bool {
	return b.HasVideo() && b.Image.IsZero()
}

func (b MediaLibraryStorage) HasVideo() bool {
	return b.Video != "" || (b.HasFile() && b.IsVideo())
}

func (b *MediaLibraryStorage) Init(site core.SiteInterface, field *aorm.Field) {
	b.Image.Init(site, field)
	b.GetOrSetFieldOption().ParseFieldTag("media_library", &field.Tag)
}

func (mls MediaLibraryStorage) Value() (driver.Value, error) {
	return mls.DBValue(&mls)
}

func (mls MediaLibraryStorage) Export(ctx *core.Context) (string, error) {
	if mls.IsZero() {
		return "", nil
	}
	if mls.Storage() != nil {
		var ep = helpers.GetStorageEndpointFromContext(ctx, mls.Storage())
		mls.Url = media.MediaURL(ep, mls.Url)
	}
	results, err := json.Marshal(mls)
	return string(results), err
}

func (mls MediaLibraryStorage) ConfigureQorMetaBeforeInitialize(metaor resource.Metaor) {
	if meta, ok := metaor.(*admin.Meta); ok {
		if meta.Type == "" {
			meta.Type = "media_library"
		}
		meta.Inline = true
		mls.Image.ConfigureQorMetaBeforeInitialize(meta)
	}
}

func (mls *MediaLibraryStorage) ScanBytes(ctx *media.Context, data []byte) (err error) {
	mls.MediaLibraryStorageAttributes = MediaLibraryStorageAttributes{}
	if err = mls.Image.ScanBytes(ctx, data); err == nil {
		err = json.Unmarshal(data, &mls.MediaLibraryStorageAttributes)
	}
	return
}



func (mls *MediaLibraryStorage) Scan(data interface{}) (err error) {
	return mls.MediaScan(media.NewContext(mls), data)
}