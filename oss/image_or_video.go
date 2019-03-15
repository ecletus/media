package oss

import (
	"github.com/aghape/media/reader_provider"
)

type ImageOrVideoType string

const (
	MEDIA_VIDEO_FILE MediaType = "video_file"
	MEDIA_VIDEO_LINK MediaType = "video_link"
)

type ImageOrLinkOrVideoLink struct {
	ImageOrLink
	VideoLink string
}

func (iv *ImageOrLinkOrVideoLink) VideoLinkProvider() reader_provider.MediaReaderProvider {
	panic("not implemented")
}

func (iv *ImageOrLinkOrVideoLink) IsVideoLink(has ...bool) bool {
	return ((has != nil && !has[0]) || iv.HasVideoLink()) && iv.MediaType == MEDIA_VIDEO_LINK
}

func (iv *ImageOrLinkOrVideoLink) HasVideoLink() bool {
	return iv.VideoLink != ""
}

type ImageOrVideoWithLink struct {
	ImageOrLinkOrVideoLink
	VideoFile Video
}

func (iv *ImageOrVideoWithLink) IsVideoFile() bool {
	return iv.HasVideoFile() && iv.MediaType == MEDIA_VIDEO_FILE
}

func (iv *ImageOrVideoWithLink) HasVideoFile() bool {
	return !iv.VideoFile.IsZero()
}
