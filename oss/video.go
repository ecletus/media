package oss

import "github.com/ecletus/media"

type Video struct {
	OSS
}

func (Video) FileExts() []string {
	return media.VideoFormats()
}
