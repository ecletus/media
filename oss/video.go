package oss

import "github.com/aghape/media"

type Video struct {
	OSS
}

func (Video) FileExts() []string {
	return media.VideoFormats()
}
