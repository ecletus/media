package media

type AcceptTypes interface {
	FileTypes() []string
}

type AcceptExts interface {
	FileExts() []string
}

type MaxSize interface {
	MaxSize() uint64
}
