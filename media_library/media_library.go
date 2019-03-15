package media_library

import (
	"encoding/json"

	"github.com/aghape/admin"
	"github.com/aghape/core"
	"github.com/aghape/core/db"
	"github.com/aghape/core/resource"
	"github.com/aghape/media/oss"
	"github.com/moisespsena-go/aorm"
)

func init() {
	aorm.StructFieldMethodCallbacks.RegisterFieldType(&MediaLibraryStorage{})
	aorm.StructFieldMethodCallbacks.RegisterFieldType(&File{})
}

type MediaOption struct {
	Video        string                     `json:",omitempty"`
	FileName     string                     `json:",omitempty"`
	URL          string                     `json:",omitempty"`
	OriginalURL  string                     `json:",omitempty"`
	CropOptions  map[string]*oss.CropOption `json:",omitempty"`
	Sizes        map[string]*oss.Size       `json:",omitempty"`
	SelectedType string                     `json:",omitempty"`
	Description  string                     `json:",omitempty"`
	Crop         bool
}

type MediaLibraryInterface interface {
	ScanMediaOptions(MediaOption) error
	SetSelectedType(string)
	GetSelectedType() string
	GetMediaOption(ctx *core.Context) MediaOption
	Init(site core.SiteInterface)
}

type QorMediaLibrary struct {
	aorm.KeyStringSerial
	SelectedType string
	File         MediaLibraryStorage `sql:"type:text;" media_library:"url:/system/{{class}}/{{primary_key}}/{{column}}.{{extension}}"`
}

func (mediaLibrary *QorMediaLibrary) Init(site core.SiteInterface) {
	mediaLibrary.File.Init(site, db.FieldCache.Get(mediaLibrary, "File"))
}

func (mediaLibrary *QorMediaLibrary) ScanMediaOptions(mediaOption MediaOption) error {
	if bytes, err := json.Marshal(mediaOption); err == nil {
		return mediaLibrary.File.Scan(bytes)
	} else {
		return err
	}
}

func (mediaLibrary *QorMediaLibrary) GetMediaOption(ctx *core.Context) MediaOption {
	return MediaOption{
		Video:        mediaLibrary.File.Video,
		FileName:     mediaLibrary.File.FileName,
		URL:          mediaLibrary.File.URL(),
		OriginalURL:  mediaLibrary.File.URL(oss.IMAGE_STYLE_ORIGNAL),
		CropOptions:  mediaLibrary.File.CropOptions,
		Sizes:        mediaLibrary.File.GetSizes(),
		SelectedType: mediaLibrary.File.SelectedType,
		Description:  mediaLibrary.File.Description,
	}
}

func (mediaLibrary *QorMediaLibrary) SetSelectedType(typ string) {
	mediaLibrary.SelectedType = typ
}

func (mediaLibrary *QorMediaLibrary) GetSelectedType() string {
	return mediaLibrary.SelectedType
}

func (QorMediaLibrary) ConfigureQorResource(res resource.Resourcer) {
	if res, ok := res.(*admin.Resource); ok {
		res.UseTheme("grid")
		res.UseTheme("media_library")
		res.IndexAttrs("File")
	}
}
