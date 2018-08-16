package media

import (
	"fmt"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/disintegration/imaging"
	"github.com/aghape/aghape/utils"
)

func getImageFormat(url string) (*imaging.Format, error) {
	formats := map[string]imaging.Format{
		".jpg":  imaging.JPEG,
		".jpeg": imaging.JPEG,
		".png":  imaging.PNG,
		".tif":  imaging.TIFF,
		".tiff": imaging.TIFF,
		".bmp":  imaging.BMP,
		".gif":  imaging.GIF,
	}

	ext := strings.ToLower(regexp.MustCompile(`(\?.*?$)`).ReplaceAllString(filepath.Ext(url), ""))
	if f, ok := formats[ext]; ok {
		return &f, nil
	}
	return nil, imaging.ErrUnsupportedFormat
}

// IsImageFormat check filename is image or not
func IsImageFormat(name string) bool {
	_, err := getImageFormat(name)
	return err == nil
}

// IsVideoFormat check filename is video or not
func IsVideoFormat(name string) bool {
	formats := []string{".mp4", ".m4p", ".m4v", ".m4v", ".mov", ".mpeg", ".webm", ".avi", ".ogg", ".ogv"}

	ext := strings.ToLower(regexp.MustCompile(`(\?.*?$)`).ReplaceAllString(filepath.Ext(name), ""))

	for _, format := range formats {
		if format == ext {
			return true
		}
	}

	return false
}

func IsSVGFormat(name string) bool {
	formats := []string{".svg", ".svgz"}

	ext := strings.ToLower(regexp.MustCompile(`(\?.*?$)`).ReplaceAllString(filepath.Ext(name), ""))

	for _, format := range formats {
		if format == ext {
			return true
		}
	}

	return false
}

func parseTagOption(str string) *Option {
	option := Option(utils.ParseTagOption(str))
	return &option
}

func MediaURL(base string, parts ...string) string {
	base = strings.TrimSuffix(base, "/")

	for _, prefix := range []string{"https://", "http://"} {
		base = strings.TrimPrefix(base, prefix)
	}

	base = "//" + base

	if len(parts) == 0 {
		return base
	}

	base += "/"

	for i, p := range parts {
		parts[i] = strings.Trim(p, "/")
	}

	url := strings.Join(parts, "/")
	return base + url
}

func MediaStyleURL(url, style string) string {
	ext := path.Ext(url)
	return fmt.Sprintf("%v.%v%v", strings.TrimSuffix(url, ext), style, ext)
}