package oss

import (
	"database/sql/driver"
	"encoding/json"
	"image"
	"mime/multipart"
	"os"
	"strings"

	"github.com/ecletus/media/reader_provider"

	"github.com/moisespsena-go/aorm"

	"github.com/ecletus/core/utils"

	"github.com/ecletus/admin"
	"github.com/ecletus/core"
	"github.com/ecletus/core/helpers"
	"github.com/ecletus/core/resource"
	"github.com/ecletus/media"
)

func init() {
	aorm.StructFieldMethodCallbacks.RegisterFieldType(&Image{})
}

var KEY_ICON_STYLE = PKG + ".iconStyle"
var META_PREVIEW = PKG + ".preview"

const (
	IMAGE_STYLE_PREVIEW = "@qor_preview"
	IMAGE_STYLE_ORIGNAL = "original"
)

var E_DEFAULT_IMAGE_URL = PKG + ".defaultImageURL"

type ImageInterface interface {
	OSSInterface
	GetSizes() map[string]*Size

	NeedCrop() bool
	Cropped(values ...bool) bool
	GetCropOption(name string) *CropOption
	GetCropOptions() map[string]*CropOption
	GetOriginalSize() *Size
	OriginalSizeDefined() bool
	Cropable() bool
}

type Image struct {
	OSS
	CropOptions  map[string]*CropOption `json:",omitempty"`
	Crop         bool                   `json:"-"`
	cropped      bool
	Sizes        map[string]*Size `json:",omitempty"`
	OriginalSize Size
	notSqlScan   bool
}

func (img *Image) GetOriginalSize() *Size {
	return &img.OriginalSize
}

// Cropped mark the image to be cropped
func (img *Image) Cropped(values ...bool) (result bool) {
	result = img.cropped
	for _, value := range values {
		img.cropped = value
	}
	return result
}

// NeedCrop return the file needs to be cropped or not
func (img *Image) NeedCrop() bool {
	return img.Crop
}

func (img Image) Cropable() bool {
	return true
}

// GetCropOption get crop options
func (img *Image) GetCropOption(name string) *CropOption {
	return img.CropOptions[strings.Split(name, "@")[0]]
}

func (img *Image) GetCropOptions() map[string]*CropOption {
	if img.CropOptions == nil {
		return map[string]*CropOption{}
	}
	return img.CropOptions
}

func (img Image) OriginalSizeDefined() bool {
	if img.Sizes != nil {
		if _, ok := img.Sizes[IMAGE_STYLE_ORIGNAL]; ok {
			return true
		}
	}
	return false
}

func (img Image) GetSizes() map[string]*Size {
	var sizes = map[string]*Size{
		IMAGE_STYLE_PREVIEW: {Width: 200, Height: 200},
	}

	if img.Sizes != nil {
		for key, value := range img.Sizes {
			sizes[key] = value
		}
	}

	return sizes
}

func (img Image) HasImage() bool {
	return img.HasFile() && img.IsImage()
}

func (img Image) SystemNames() []string {
	names := img.OSS.SystemNames()
	return append(names, IMAGE_STYLE_PREVIEW, IMAGE_STYLE_ORIGNAL)
}

func (img *Image) Export(ctx *core.Context) (string, error) {
	if img.Storage() != nil {
		var url = helpers.GetStorageEndpointFromContext(ctx, img.Storage())
		img.Url = media.MediaURL(url, img.Url)
	}
	results, err := json.Marshal(img)
	return string(results), err
}

func (img Image) Value() (driver.Value, error) {
	return img.DBValue(&img)
}

func (img *Image) Scan(data interface{}) (err error) {
	return img.MediaScan(media.NewContext(img), data)
}

func (img *Image) ContextScan(ctx *core.Context, data interface{}) (err error) {
	img.notSqlScan = true
	return img.Set(media.NewContext(img), data)
}

func (img *Image) Set(ctx *media.Context, data interface{}) (err error) {
	if !img.isNew {
		defer func() {
			if err == nil && img.isNew {
				// reset original size
				img.OriginalSize = Size{}
				if img.Sizes != nil {
					img.Sizes = nil
				}
			}
		}()
	}
	return img.OSS.Set(ctx, data)
}

func (img *Image) ScanBytes(ctx *media.Context, data []byte) (err error) {
	if img.Sizes == nil {
		img.Sizes = map[string]*Size{}
	}
	if img.CropOptions == nil {
		img.CropOptions = map[string]*CropOption{}
	}
	err = img.OSS.ScanBytes(ctx, data)
	if err != nil {
		return
	}

	if img.HasFile() && !img.Delete && img.Cropable() {
		var imgData struct {
			CropOptions  map[string]*CropOption
			Crop         bool
			Sizes        map[string]*Size
			OriginalSize *Size
		}

		if err = json.Unmarshal(data, &imgData); err == nil {
			if imgData.Crop {
				img.Crop = true
			}

			for key, value := range imgData.CropOptions {
				img.CropOptions[key] = value
			}

			for key, value := range imgData.Sizes {
				img.Sizes[key] = value
			}

			for key, value := range img.CropOptions {
				if _, ok := img.Sizes[key]; !ok {
					img.Sizes[key] = &Size{Width: value.Width, Height: value.Height}
				}
			}

			if imgData.OriginalSize != nil {
				img.OriginalSize = *imgData.OriginalSize
			}
		}
	}
	return
}

func (img *Image) MediaScan(ctx *media.Context, data interface{}) (err error) {
	switch values := data.(type) {
	case []byte:
		return ctx.Media.ScanBytes(ctx, values)
	case string:
		err = img.MediaScan(ctx, []byte(values))
	case []string:
		for _, str := range values {
			if err = img.Scan([]byte(str)); err != nil {
				return err
			}
		}
	case *os.File:
		img.OriginalSize = Size{}
		img.Sizes = nil
		img.CropOptions = nil
		return img.OSS.MediaScan(ctx, data)
	case *multipart.FileHeader:
		img.OriginalSize = Size{}
		img.Sizes = nil
		img.CropOptions = nil
		return img.OSS.MediaScan(ctx, data)
	case []*multipart.FileHeader:
		if len(values) > 0 {
			return img.MediaScan(ctx, values[0])
		}
	default:
		return img.OSS.MediaScan(ctx, data)
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

func (img *Image) Init(site core.SiteInterface, field *aorm.Field) {
	img.OSS.Init(site, field)
	img.GetOrSetFieldOption().ParseFieldTag("image", &field.Tag)
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

func ImageMetaOnDefaultValue(meta *admin.Meta, cb func(e *admin.MetaValuerEvent)) {
	admin.OnMetaValue(meta, E_DEFAULT_IMAGE_URL, cb)
}

func ImageMetaGetDefaultURL(meta *admin.Meta, recorde interface{}, ctx *core.Context) (url string) {
	return meta.TriggerValuerEvent(E_DEFAULT_IMAGE_URL, recorde, ctx, nil, "").(string)
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
	urlMeta := meta.BaseResource.SetMeta(&admin.Meta{
		Name: meta.Name + key,
		Type: "image",
		Valuer: func(recorde interface{}, ctx *core.Context) interface{} {
			if basic, ok := recorde.(resource.BasicIcon); ok {
				return basic.BasicIcon()
			}
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

// Size is a struct, used for `GetSizes` method, it will return a slice of Size, media library will crop images automatically based on it
type Size struct {
	Width  int
	Height int
}

// CropOption includes crop options
type CropOption struct {
	X, Y, Width, Height int
}

func (cropOption CropOption) Rectangle() *image.Rectangle {
	return &image.Rectangle{
		Min: image.Point{X: cropOption.X, Y: cropOption.Y},
		Max: image.Point{X: cropOption.X + cropOption.Width, Y: cropOption.Y + cropOption.Height},
	}
}

type MediaType string

const (
	MEDIA_IMAGE_FILE MediaType = "image_file"
	MEDIA_IMAGE_LINK MediaType = "image_link"
)

type ImageWithLink struct {
	ImageLink string
}

func (i *ImageWithLink) HasImageLink() bool {
	return i.ImageLink != ""
}

type ImageOrLink struct {
	MediaType MediaType `sql:"type:varchar(24)"`
	ImageFile Image
	ImageWithLink
}

func (i *ImageOrLink) IsImageFile(has ...bool) bool {
	return ((has != nil && !has[0]) || i.HasImageFile()) && (i.MediaType == "" || i.MediaType == MEDIA_IMAGE_FILE)
}

func (i *ImageOrLink) IsImageLink(has ...bool) bool {
	return ((has != nil && !has[0]) || i.HasImageLink()) && i.MediaType == MEDIA_IMAGE_LINK
}

func (i *ImageOrLink) HasImageFile() bool {
	return !i.ImageFile.IsZero()
}

func (i *ImageOrLink) ImageURLProvider() reader_provider.MediaReaderProvider {
	panic("not implemented")
}
