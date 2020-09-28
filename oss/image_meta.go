package oss

import (
	"reflect"

	"github.com/ecletus/core"
	"github.com/ecletus/core/resource"

	"github.com/ecletus/admin"
)

type ImageMeta struct {
	FormattedStyle string
	URLFunc        func(meta *admin.Meta, ctx *core.Context, record interface{}, value *Image) string
	SvgDisabled    bool
	meta           *admin.Meta
}

func (this *ImageMeta) DefaultURL(ctx *core.Context, record interface{}, value *Image) string {
	if value.IsZero() {
		return ""
	}
	if value.IsImage() {
		if this.FormattedStyle != "" {
			if _, ok := value.Sizes[this.FormattedStyle]; ok {
				return value.FullURL(ctx, this.FormattedStyle)
			}
		}
		return value.FullURL(ctx, IMAGE_STYLE_PREVIEW)
	} else if value.IsSVG() {
		return value.FullURL(ctx)
	}
	return ""
}

func (this *ImageMeta) GetURL(ctx *core.Context, record interface{}, value *Image) (url string) {
	if this.URLFunc != nil {
		if url = this.URLFunc(this.meta, ctx, record, value); url != "" {
			return
		}
	}
	return this.DefaultURL(ctx, record, value)
}

func (this *ImageMeta) StyleURL(ctx *core.Context, record interface{}, style ...string) (url string) {
	value := this.meta.Value(ctx, record)
	if value == nil {
		return
	}
	var img *Image
	switch t := value.(type) {
	case Image:
		img = &t
	case *Image:
		img = t
	default:
		return ""
	}
	for _, style := range style {
		if _, ok := img.Sizes[style]; ok {
			return img.FullURL(ctx, style)
		}
	}
	return img.FullURL(ctx, IMAGE_STYLE_ORIGNAL)
}

func (this *ImageMeta) ConfigureQorMeta(metaor resource.Metaor) {
	if this.meta != nil {
		return
	}
	this.meta = metaor.(*admin.Meta)
	if this.meta.Type == "" {
		this.meta.Type = "image"
	}
	if !this.SvgDisabled {
		this.SvgDisabled = this.meta.Tags.Flag("SVG_DISABLED")
	}

	if this.meta.FormattedValuer == nil {
		if this.meta.Typ.Kind() == reflect.Ptr {
			this.meta.SetFormattedValuer(func(recorde interface{}, ctx *core.Context) interface{} {
				value := this.meta.Value(ctx, recorde)
				if value == nil {
					return nil
				}
				return this.GetURL(ctx, recorde, value.(*Image))
			})
		} else {
			this.meta.SetFormattedValuer(func(recorde interface{}, ctx *core.Context) interface{} {
				value := this.meta.Value(ctx, recorde)
				if value == nil {
					return nil
				}
				img := value.(Image)
				return this.GetURL(ctx, recorde, &img)
			})
		}
	}
}

func init() {
	admin.RegisterMetaTypeConfigor(reflect.TypeOf(Image{}), func(meta *admin.Meta) {
		if meta.Config == nil {
			meta.Config = &ImageMeta{}
		}
		meta.Config.ConfigureQorMeta(meta)
	})
}
