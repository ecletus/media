package media

import (
	"bytes"
	"image"
	"image/draw"
	"image/gif"
	"image/png"
	_ "image/jpeg"
	"io/ioutil"
	"mime/multipart"
	"github.com/disintegration/imaging"
)

var mediaHandlers = make(map[string]MediaHandler)

// MediaHandler media library handler interface, defined which files could be handled, and the handler
type MediaHandler interface {
	CouldHandle(media Media) bool
	Handle(media Media, file multipart.File, option *Option) error
}

// RegisterMediaHandler register Media library handler
func RegisterMediaHandler(name string, handler MediaHandler) {
	mediaHandlers[name] = handler
}

// imageHandler default image handler
type imageHandler struct{}

func (imageHandler) CouldHandle(media Media) bool {
	return media.IsImage()
}

func init() {
	// damn important or else At(), Bounds() functions will
	// caused memory pointer error!!
	image.RegisterFormat("png", "png", png.Decode, png.DecodeConfig)
}

func (imageHandler) Handle(media Media, file multipart.File, option *Option) (err error) {
	var fileBuffer bytes.Buffer
	if fileBytes, err := ioutil.ReadAll(file); err == nil {
		if media.Old() != nil {
			media.Old().AddName("original")
			_, err = media.RemoveOld()
			if err != nil {
				return err
			}
		}

		fileBuffer.Write(fileBytes)

		if err = media.Store(media.URL("original"), &fileBuffer); err == nil {
			file.Seek(0, 0)

			if format, err := getImageFormat(media.URL()); err == nil {
				if *format == imaging.GIF {
					var buffer bytes.Buffer
					if g, err := gif.DecodeAll(file); err == nil {
						if cropOption := media.GetCropOption("original"); cropOption != nil {
							for i := range g.Image {
								img := imaging.Crop(g.Image[i], *cropOption)
								g.Image[i] = image.NewPaletted(img.Rect, g.Image[i].Palette)
								draw.Draw(g.Image[i], img.Rect, img, image.Pt(0, 0), draw.Src)
								if i == 0 {
									g.Config.Width = img.Rect.Dx()
									g.Config.Height = img.Rect.Dy()
								}
							}
						}

						gif.EncodeAll(&buffer, g)
						media.Store(media.URL(), &buffer)
					} else {
						return err
					}

					// save sizes image
					for key, size := range media.GetSizes() {
						file.Seek(0, 0)
						if g, err := gif.DecodeAll(file); err == nil {
							for i := range g.Image {
								var img image.Image = g.Image[i]
								if cropOption := media.GetCropOption(key); cropOption != nil {
									img = imaging.Crop(g.Image[i], *cropOption)
								}
								img = imaging.Thumbnail(img, size.Width, size.Height, imaging.Lanczos)
								g.Image[i] = image.NewPaletted(image.Rect(0, 0, size.Width, size.Height), g.Image[i].Palette)
								draw.Draw(g.Image[i], image.Rect(0, 0, size.Width, size.Height), img, image.Pt(0, 0), draw.Src)
							}

							var result bytes.Buffer
							g.Config.Width = size.Width
							g.Config.Height = size.Height
							gif.EncodeAll(&result, g)
							media.Store(media.URL(key), &result)
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

					if img, err := imaging.Decode(file); err == nil {
						// save original image
						if cropOption := media.GetCropOption("original"); cropOption != nil {
							img = imaging.Crop(img, *cropOption)
						}

						// Save default image
						var buffer bytes.Buffer
						imaging.Encode(&buffer, img, *format)
						media.Store(media.URL(), &buffer)

						// save sizes image
						for key, size := range media.GetSizes() {
							newImage := img
							if cropOption := media.GetCropOption(key); cropOption != nil {
								newImage = imaging.Crop(newImage, *cropOption)
							}

							dst := imaging.Thumbnail(newImage, size.Width, size.Height, imaging.Lanczos)
							var buffer bytes.Buffer
							imaging.Encode(&buffer, dst, *format)
							media.Store(media.URL(key), &buffer)
						}
					} else {
						return err
					}
				}
			}
		} else {
			return err
		}
	}

	return err
}

func init() {
	RegisterMediaHandler("image_handler", imageHandler{})
}
