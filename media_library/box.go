package media_library

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"mime/multipart"
	"reflect"

	"github.com/ecletus/media/oss"

	"github.com/moisespsena-go/aorm"

	"github.com/ecletus/admin"
	"github.com/ecletus/core"
	"github.com/ecletus/core/resource"
	"github.com/ecletus/core/utils"
	"github.com/ecletus/media"
)

type MediaBox struct {
	Values string `json:"-" gorm:"size:4294967295;"`
	Files  []File `json:",omitempty"`
}

func (mediaBox MediaBox) URL(styles ...string) string {
	for _, file := range mediaBox.Files {
		return file.URL(styles...)
	}
	return ""
}

func (mediaBox *MediaBox) Scan(data interface{}) (err error) {
	switch values := data.(type) {
	case []byte:
		if mediaBox.Values = string(values); mediaBox.Values != "" {
			return json.Unmarshal(values, &mediaBox.Files)
		}
	case string:
		return mediaBox.Scan([]byte(values))
	case []string:
		for _, str := range values {
			if err := mediaBox.Scan(str); err != nil {
				return err
			}
		}
	}
	return nil
}

func (mediaBox MediaBox) Value() (driver.Value, error) {
	if len(mediaBox.Files) > 0 {
		return json.Marshal(mediaBox.Files)
	}
	return mediaBox.Values, nil
}

func (mediaBox MediaBox) ConfigureQorMeta(metaor resource.Metaor) {
	if meta, ok := metaor.(*admin.Meta); ok {
		if meta.Config == nil {
			meta.Config = &MediaBoxConfig{}
		}

		if meta.FormattedValuer == nil {
			meta.FormattedValuer = func(record interface{}, context *core.Context) interface{} {
				if mediaBox, ok := meta.GetValuer()(record, context).(*MediaBox); ok {
					return mediaBox.URL()
				}
				return ""
			}
			meta.SetFormattedValuer(meta.FormattedValuer)
		}

		if config, ok := meta.Config.(*MediaBoxConfig); ok {
			res := meta.GetBaseResource().(*admin.Resource)
			if config.RemoteDataResource == nil {
				dataResource := res.GetOrParentResourceByID("QorMediaLibrary")
				if dataResource == nil {
					panic("Imposible to auto detect RemoteDataResource \"QorMediaLibrary\".")
				}
				config.RemoteDataResource = admin.NewDataResource(dataResource)
			}

			if _, ok := config.RemoteDataResource.Resource.Value.(MediaLibraryInterface); !ok {
				panic(fmt.Errorf("%v havn't implement MediaLibraryInterface, please fix that.",
					reflect.TypeOf(config.RemoteDataResource.Resource.Value)))
			}

			config.RemoteDataResource.Resource.Meta(&admin.Meta{
				Name: "MediaOption",
				Type: "hidden",
				Setter: func(record interface{}, metaValue *resource.MetaValue, context *core.Context) error {
					if mediaLibrary, ok := record.(MediaLibraryInterface); ok {
						mediaLibrary.Init(context.Site)
						var mediaOption MediaOption
						if err := json.Unmarshal([]byte(utils.ToString(metaValue.Value)), &mediaOption); err == nil {
							mediaOption.FileName = ""
							mediaOption.URL = ""
							mediaOption.OriginalURL = ""
							mediaLibrary.ScanMediaOptions(mediaOption)
						}
					}
					return nil
				},
				Valuer: func(record interface{}, context *core.Context) interface{} {
					if mediaLibrary, ok := record.(MediaLibraryInterface); ok {
						if value, err := json.Marshal(mediaLibrary.GetMediaOption(context)); err == nil {
							return string(value)
						}
					}
					return ""
				},
			})

			config.RemoteDataResource.Resource.Meta(&admin.Meta{
				Name: "SelectedType",
				Type: "hidden",
				Valuer: func(record interface{}, context *core.Context) interface{} {
					if mediaLibrary, ok := record.(MediaLibraryInterface); ok {
						return mediaLibrary.GetSelectedType()
					}
					return ""
				},
			})

			config.RemoteDataResource.Resource.AddProcessor(func(record interface{}, metaValues *resource.MetaValues, context *core.Context) error {
				if mediaLibrary, ok := record.(MediaLibraryInterface); ok {
					mediaLibrary.Init(context.Site)
					var filename string
					var mediaOption MediaOption

					for _, metaValue := range metaValues.Values {
						if fileHeaders, ok := metaValue.Value.([]*multipart.FileHeader); ok {
							for _, fileHeader := range fileHeaders {
								filename = fileHeader.Filename
							}
						}
					}

					if metaValue := metaValues.Get("MediaOption"); metaValue != nil {
						mediaOptionStr := utils.ToString(metaValue.Value)
						json.Unmarshal([]byte(mediaOptionStr), &mediaOption)
					}

					if mediaOption.SelectedType == "video_link" {
						mediaLibrary.SetSelectedType("video_link")
					} else if filename != "" {
						if media.IsImageFormat(filename) {
							mediaLibrary.SetSelectedType("image")
						} else if media.IsVideoFormat(filename) {
							mediaLibrary.SetSelectedType("video")
						} else {
							mediaLibrary.SetSelectedType("file")
						}
					}
				}
				return nil
			})

			config.RemoteDataResource.Resource.UseTheme("grid")
			config.RemoteDataResource.Resource.UseTheme("media_library")
			if config.RemoteDataResource.Resource.Config.PageCount == 0 {
				config.RemoteDataResource.Resource.Config.PageCount = admin.PaginationPageCount / 2 * 3
			}
			config.RemoteDataResource.Resource.IndexAttrs(config.RemoteDataResource.Resource.IndexAttrs(), "-MediaOption")
			config.RemoteDataResource.Resource.NewAttrs(config.RemoteDataResource.Resource.NewAttrs(), "MediaOption")
			config.RemoteDataResource.Resource.EditAttrs(config.RemoteDataResource.Resource.EditAttrs(), "MediaOption")

			config.SelectManyConfig.RemoteDataResource = config.RemoteDataResource
			config.SelectManyConfig.ConfigureQorMeta(meta)
		}

		meta.Type = "media_box"
	}
}

func (mediaBox MediaBox) Crop(context *core.Context, res *admin.Resource, db *aorm.DB, mediaOption MediaOption) (err error) {
	for _, file := range mediaBox.Files {
		var ID aorm.ID
		if ID, err = res.ModelStruct.DefaultID().SetValue(file.ID); err != nil {
			return
		}
		context := (&core.Context{ResourceID: ID}).SetRawDB(db)
		record := res.NewStruct(context.Site)
		crud := res.Crud(context)
		if err = crud.FindOne(record); err == nil {
			if mediaLibrary, ok := record.(MediaLibraryInterface); ok {
				mediaOption.Crop = true
				if err = mediaLibrary.ScanMediaOptions(mediaOption); err == nil {
					err = crud.Update(record)
				}
			} else {
				err = errors.New("invalid media library resource")
			}
		}
		if err != nil {
			return
		}
	}
	return
}

// MediaBoxConfig configure MediaBox metas
type MediaBoxConfig struct {
	RemoteDataResource *admin.DataResource
	Sizes              map[string]*oss.Size
	Max                uint
	admin.SelectManyConfig
}

func (*MediaBoxConfig) ConfigureQorMeta(resource.Metaor) {
}

func (*MediaBoxConfig) GetTemplate(context *admin.Context, metaType string) ([]byte, error) {
	return nil, errors.New("not implemented")
}
