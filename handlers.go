package media

import (
	_ "image/jpeg"
	"mime/multipart"

	"github.com/moisespsena/go-error-wrap"
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

func MediaHandlers() map[string]MediaHandler {
	return mediaHandlers
}

func Handle(media Media, cb func(name string, handle MediaHandler) error) (err error) {
	for name, handler := range mediaHandlers {
		if handler.CouldHandle(media) {
			if err = cb(name, handler); err != nil {
				return errwrap.Wrap(err, "Media Handler %q", name)
			}
		}
	}
	return
}
