package media_library

import (
	"fmt"
	"path"
	"strings"

	"github.com/moisespsena-go/bid"

	"github.com/ecletus/media"
)

type File struct {
	ID          bid.BID
	Url         string
	VideoLink   string
	FileName    string
	Description string
}

// IsImage return if it is an image
func (f File) IsImage() bool {
	return media.IsImageFormat(f.Url)
}

func (f File) IsVideo() bool {
	return media.IsVideoFormat(f.Url)
}

func (f File) IsSVG() bool {
	return media.IsSVGFormat(f.Url)
}

func (file File) URL(styles ...string) string {
	if file.Url != "" && len(styles) > 0 {
		ext := path.Ext(file.Url)
		return fmt.Sprintf("%v.%v%v", strings.TrimSuffix(file.Url, ext), styles[0], ext)
	}
	return file.Url
}
