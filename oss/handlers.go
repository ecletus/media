package oss

import (
	"bytes"
	"image"
	"image/draw"
	"image/gif"
	_ "image/jpeg"
	"image/png"
	"mime/multipart"

	"github.com/aghape/media"

	"github.com/disintegration/imaging"
)

// imageHandler default image handler
type imageHandler struct{}

func (imageHandler) CouldHandle(m media.Media) bool {
	if m.IsImage() {
		if im, ok := m.(ImageInterface); ok {
			return im.NeedCrop()
		}
	}
	return false
}

func init() {
	// damn important or else At(), Bounds() functions will
	// caused memory pointer error!!
	image.RegisterFormat("png", "png", png.Decode, png.DecodeConfig)
}

func (imageHandler) Handle(m media.Media, file multipart.File, option *media.Option) (err error) {
	im := m.(ImageInterface)
	if format, err := media.GetImageFormat(im.URL()); err == nil {
		var buffer bytes.Buffer
		if *format == imaging.GIF {
			if g, err := gif.DecodeAll(file); err == nil {
				if cropOption := im.GetCropOption(IMAGE_STYLE_ORIGNAL); cropOption != nil {
					for i := range g.Image {
						img := imaging.Crop(g.Image[i], *cropOption.Rectangle())
						g.Image[i] = image.NewPaletted(img.Rect, g.Image[i].Palette)
						draw.Draw(g.Image[i], img.Rect, img, image.Pt(0, 0), draw.Src)
						if i == 0 {
							g.Config.Width = img.Rect.Dx()
							g.Config.Height = img.Rect.Dy()
						}
					}
				}

				gif.EncodeAll(&buffer, g)
				im.Store(im.URL(), &buffer)
			} else {
				return err
			}

			// save sizes image
			for key, size := range im.GetSizes() {
				file.Seek(0, 0)
				if g, err := gif.DecodeAll(file); err == nil {
					for i := range g.Image {
						var img image.Image = g.Image[i]
						if cropOption := im.GetCropOption(key); cropOption != nil {
							img = imaging.Crop(g.Image[i], *cropOption.Rectangle())
						}
						img = imaging.Thumbnail(img, size.Width, size.Height, imaging.Lanczos)
						g.Image[i] = image.NewPaletted(image.Rect(0, 0, size.Width, size.Height), g.Image[i].Palette)
						draw.Draw(g.Image[i], image.Rect(0, 0, size.Width, size.Height), img, image.Pt(0, 0), draw.Src)
					}

					var result bytes.Buffer
					g.Config.Width = size.Width
					g.Config.Height = size.Height
					gif.EncodeAll(&result, g)
					im.Store(im.URL(key), &result)
				}
			}
		} else {
			/*tmp, err := os.Create("tmp.png")
			defer tmp.Close()
			io.Copy(tmp, file)

			_, err = png.Decode(file)
			if err != nil {
				fmt.Println("PNG DECODE ERROR: ", err)
				fmt.Println("PNG DECODE ERROR: ", file.Read)
				return err
			}
			*/
			///return toNRGBA(img), nil

			if img, err := imaging.Decode(&buffer); err == nil {
				// save original image
				if cropOption := im.GetCropOption(IMAGE_STYLE_ORIGNAL); cropOption != nil {
					img = imaging.Crop(img, *cropOption.Rectangle())
				}

				// Save default image
				imaging.Encode(&buffer, img, *format)
				im.Store(im.URL(IMAGE_STYLE_ORIGNAL), &buffer)

				// save sizes image
				for key, size := range im.GetSizes() {
					if key == IMAGE_STYLE_ORIGNAL {
						continue
					}
					newImage := img
					if cropOption := im.GetCropOption(key); cropOption != nil {
						newImage = imaging.Crop(newImage, *cropOption.Rectangle())
					}

					dst := imaging.Thumbnail(newImage, size.Width, size.Height, imaging.Lanczos)
					var buffer bytes.Buffer
					imaging.Encode(&buffer, dst, *format)
					im.Store(im.URL(key), &buffer)
				}
			} else {
				return err
			}
		}
	}

	return err
}

func init() {
	media.RegisterMediaHandler("image_handler", imageHandler{})
}
