package oss

import (
	"database/sql/driver"
	"encoding/json"
	"strings"

	"github.com/moisespsena-go/aorm"

	"github.com/aghape/core/utils"

	"github.com/aghape/admin"
	"github.com/aghape/core"
	"github.com/aghape/core/resource"
	"github.com/aghape/media"
)

func init() {
	aorm.StructFieldMethodCallbacks.RegisterFieldType(&Image{})
}

var KEY_ICON_STYLE = PKG + ".iconStyle"
var META_PREVIEW = PKG + ".preview"

const IMAGE_STYLE_PREVIEW = "@qor_preview"

var E_DEFAULT_IMAGE_URL = PKG + ".defaultImageURL"

type Image struct {
	OSS
	Sizes map[string]*media.Size `json:",omitempty"`
}

func (img Image) Names() []string {
	names := img.OSS.Names()
	return append(names, IMAGE_STYLE_PREVIEW)
}

func (img Image) GetSizes() map[string]*media.Size {
	if len(img.Sizes) == 0 && !(img.GetFileHeader() != nil || img.Crop) {
		return map[string]*media.Size{}
	}

	var sizes = map[string]*media.Size{
		IMAGE_STYLE_PREVIEW: {Width: 200, Height: 200},
	}

	for key, value := range img.Sizes {
		sizes[key] = value
	}
	return sizes
}

func (img *Image) Export() (string, error) {
	if img.Storage() != nil {
		img.Url = media.MediaURL(img.Storage().GetEndpoint(), img.Url)
	}
	results, err := json.Marshal(img)
	return string(results), err
}

func (img Image) Value() (driver.Value, error) {
	results, err := json.Marshal(img)
	return string(results), err
}

func (img *Image) ContextScan(ctx *core.Context, data interface{}) (err error) {
	return img.CallContextScan(ctx, data, img.Scan)
}

func (img *Image) Scan(data interface{}) (err error) {
	switch values := data.(type) {
	case []byte:
		if img.Sizes == nil {
			img.Sizes = map[string]*media.Size{}
		}
		if img.CropOptions == nil {
			img.CropOptions = map[string]*media.CropOption{}
		}
		cropOptions := img.CropOptions
		sizeOptions := img.Sizes

		if string(values) != "" {
			img.Base.Scan(values)

			if err = json.Unmarshal(values, img); err == nil {
				for key, value := range cropOptions {
					if _, ok := img.CropOptions[key]; !ok {
						img.CropOptions[key] = value
					}
				}

				for key, value := range sizeOptions {
					if _, ok := img.Sizes[key]; !ok {
						img.Sizes[key] = value
					}
				}

				for key, value := range img.CropOptions {
					if _, ok := img.Sizes[key]; !ok {
						img.Sizes[key] = &media.Size{Width: value.Width, Height: value.Height}
					}
				}
			}
		}
	case string:
		err = img.Scan([]byte(values))
	case []string:
		for _, str := range values {
			if err = img.Scan(str); err != nil {
				return err
			}
		}
	default:
		return img.Base.Scan(data)
	}
	return nil
}

func (img *Image) ConfigureQorMetaBeforeInitialize(metaor resource.Metaor) {
	if meta, ok := metaor.(*admin.Meta); ok {
		if meta.Type == "" {
			meta.Type = "image"
		}
		img.OSS.ConfigureQorMetaBeforeInitialize(meta)
		meta.DefaultValueFunc = func(recorde interface{}, context *core.Context) interface{} {
			return Image{}
		}
	}
}

func (Image) MaxSize() uint64 {
	return 1024 * 1024 //1MB
}

func (Image) FileTypes() []string {
	return []string{"image/jpeg", "image/png"}
}

func (Image) FileExts() []string {
	return []string{"jpg", "png"}
}

func ImageMetaOnDefaultValue(meta *admin.Meta, cb func(e *admin.MetaValueEvent)) {
	admin.OnMetaValue(meta, E_DEFAULT_IMAGE_URL, cb)
}

func ImageMetaGetDefaultURL(meta *admin.Meta, recorde interface{}, ctx *core.Context) (url string) {
	return meta.TriggerValueEvent(E_DEFAULT_IMAGE_URL, recorde, ctx, nil, "").(string)
}

func GetImageStyle(ctx *core.Context) string {
	style := ctx.URLParam("image_style")
	if style == "" {
		style = ctx.Data().GetString("image_style")
	}
	return style
}

func ImageMetaURL(meta *admin.Meta, key, defaultStyle string) *admin.Meta {
	styleKey := key + "Style"
	urlMeta := meta.Resource.SetMeta(&admin.Meta{
		Name: meta.Name + key,
		Type: "image",
		Valuer: func(recorde interface{}, ctx *core.Context) interface{} {
			var (
				style = GetImageStyle(ctx)
				v     = meta.Value(ctx, recorde)
				s     string
			)

			if style == "" {
				style = meta.Options.GetString(styleKey, defaultStyle)
			}

			if meta.IsZero(recorde, v) {
				func() {
					defer ctx.Data().With()
					ctx.Data().Set("image_style", style)
					v = ImageMetaGetDefaultURL(meta, recorde, ctx)
				}()
			}

			if meta.IsZero(recorde, v) {
				return ""
			}

			switch vt := v.(type) {
			case string:
				s = vt
			case media.FullURL:
				return vt.FullURL()
			case media.FullURLContext:
				return vt.FullURL(ctx)
			case media.FullURLStyle:
				return vt.FullURL(style)
			case media.FullURLContextStyle:
				return vt.FullURL(ctx, style)
			case media.URL:
				return vt.URL()
			case media.URLContext:
				return vt.URL(ctx)
			case media.URLStyle:
				return vt.URL(style)
			case media.URLContextStyle:
				return vt.URL(ctx, style)
			default:
				s = utils.StringifyContext(vt, ctx)
			}
			if s != "" && !strings.Contains(s, "//") {
				s = ctx.GenGlobalStaticURL(s)
			}
			return s
		},
	})
	meta.Options.Set(key+"Meta", urlMeta)
	return urlMeta
}
