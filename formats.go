package media

var (
	videoFormats = []string{".mp4", ".m4p", ".m4v", ".m4v", ".mov", ".mpeg", ".webm", ".avi", ".ogg", ".ogv"}
)

func VideoFormats() []string {
	return videoFormats[:]
}
