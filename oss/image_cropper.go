package oss

import (
	"bytes"
	"image"
	"image/draw"
	"image/gif"
	"io"

	"github.com/aghape/media"
	"github.com/moisespsena/go-error-wrap"

	"github.com/disintegration/imaging"
)

type ImageCropper struct {
	file    io.ReadSeeker
	Image   ImageInterface
	Img     image.Image
	Format  imaging.Format
	gif     *gif.GIF
	handler func(options map[string]*CropperOption, cb func(key string, f *bytes.Buffer) error) error
}

type CropperOption struct {
	Size *Size
	Crop *CropOption
}

func NewImageCropper(img ImageInterface, file io.ReadSeeker) (cropper *ImageCropper, err error) {
	var format *imaging.Format
	if format, err = media.GetImageFormat(img.URL()); err != nil {
		return
	}
	cropper = &ImageCropper{file, img, nil, *format, nil, nil}
	if *format == imaging.GIF {
		if cropper.gif, err = gif.DecodeAll(file); err != nil {
			return nil, errwrap.Wrap(err, "GIF Decode")
		}
		cropper.handler = cropper.gifHandler
		return
	}
	cropper.handler = cropper.defaultHandler
	if cropper.Img, err = imaging.Decode(file); err != nil {
		return nil, errwrap.Wrap(err, "Decode")
	}

	return
}

func (cropper *ImageCropper) Width() (w int) {
	if cropper.gif != nil {
		return cropper.gif.Config.Width
	}
	return cropper.Img.Bounds().Max.X
}

func (cropper *ImageCropper) Height() (w int) {
	if cropper.gif != nil {
		return cropper.gif.Config.Height
	}
	return cropper.Img.Bounds().Max.Y
}

func (cropper *ImageCropper) Size() (w int, h int) {
	return cropper.Width(), cropper.Height()
}

func (cropper *ImageCropper) defaultHandler(options map[string]*CropperOption, cb func(key string, f *bytes.Buffer) error) (err error) {
	w, h := cropper.Size()
	for key, opt := range options {
		img := cropper.Img
		var (
			crop   = opt.Crop != nil
			resize = opt.Size != nil && !(opt.Size.Width == w && opt.Size.Width == h)
		)
		if crop {
			img = imaging.Crop(img, *opt.Crop.Rectangle())
		}
		if resize {
			img = imaging.Thumbnail(img, opt.Size.Width, opt.Size.Height, imaging.Lanczos)
		}
		if crop || resize {
			var buffer bytes.Buffer
			if err = imaging.Encode(&buffer, img, cropper.Format); err != nil {
				return errwrap.Wrap(err, "Encode %q", key)
			}
			if err = cb(key, &buffer); err != nil {
				return errwrap.Wrap(err, "Callback of %q", key)
			}
		}
	}
	return
}

func (cropper *ImageCropper) newGif() *gif.GIF {
	g := *cropper.gif
	g.Image = cropper.gif.Image[:]
	return &g
}

func (cropper *ImageCropper) gifHandler(options map[string]*CropperOption, cb func(key string, f *bytes.Buffer) error) (err error) {
	for key, cropOption := range options {
		g := cropper.newGif()
		rectangle := image.Rect(0, 0, g.Config.Width, g.Config.Height)
		if cropOption.Crop != nil {
			rectangle = *cropOption.Crop.Rectangle()
		}
		if cropOption.Size != nil {
			rectangle = image.Rect(0, 0, cropOption.Size.Width, cropOption.Size.Height)
		}

		for i := range g.Image {
			var img image.Image
			img = g.Image[i]
			if cropOption.Crop != nil {
				img = imaging.Crop(img, *cropOption.Crop.Rectangle())
			}
			if cropOption.Size != nil {
				img = imaging.Thumbnail(img, cropOption.Size.Width, cropOption.Size.Height, imaging.Lanczos)
			}
			g.Image[i] = image.NewPaletted(rectangle, g.Image[i].Palette)
			draw.Draw(g.Image[i], rectangle, img, image.Pt(0, 0), draw.Src)
		}

		var result bytes.Buffer
		g.Config.Width = rectangle.Max.X
		g.Config.Height = rectangle.Max.Y
		if err = gif.EncodeAll(&result, g); err != nil {
			return errwrap.Wrap(err, "GIF EncodeAll %q", key)
		}
		if err = cb(key, &result); err != nil {
			return errwrap.Wrap(err, "GIF Callback of %q", key)
		}
	}
	return
}

func (cropper *ImageCropper) Crop(options map[string]*CropperOption, cb func(key string, f *bytes.Buffer) error) (err error) {
	return cropper.handler(options, cb)
}

func (cropper *ImageCropper) CropNames(cb func(key string, f *bytes.Buffer) error, names ...string) (err error) {
	options := map[string]*CropperOption{}
	sizes := cropper.Image.GetSizes()
	for _, name := range names {
		opt := &CropperOption{
			Crop: cropper.Image.GetCropOption(name),
			Size: sizes[name],
		}
		if opt.Crop != nil || opt.Size != nil {
			options[name] = opt
		}
	}
	if len(options) == 0 {
		return
	}
	return cropper.Crop(options, cb)
}
