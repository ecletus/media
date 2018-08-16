package media

import (
	"database/sql/driver"
	"image"
	"io"
	"os"
	"strings"

	"github.com/jinzhu/gorm"
	"github.com/aghape/aghape"
	"github.com/aghape/oss"
)

type Old struct {
	Url   string
	names map[string]int
}

func (old *Old) AddName(names ...string) {
	for _, name := range names {
		old.names[name] = 1
	}
}

func (old *Old) Names() (names []string) {
	for name, _ := range old.names {
		names = append(names, name)
	}
	return
}

// Media is an interface including methods that needs for a media library storage
type Media interface {
	Site() qor.SiteInterface
	SetSite(site qor.SiteInterface)
	Storage() oss.StorageInterface
	SetStorage(storage oss.StorageInterface)
	Init(site qor.SiteInterface, field *gorm.Field)
	FieldOption() *Option
	SetFieldOption(option *Option)

	Scan(value interface{}) error
	Value() (driver.Value, error)
	GetURL(scope *gorm.Scope, field *gorm.Field, templater URLTemplater) string

	GetFileHeader() FileHeader
	GetFileName() string

	GetSizes() map[string]*Size
	NeedCrop() bool
	Cropped(values ...bool) bool
	GetCropOption(name string) *image.Rectangle

	Store(url string, reader io.Reader) error
	Retrieve(url string) (*os.File, error)
	Remove(url string) error

	IsImage() bool

	GetURLTemplate(option *Option) string
	URL(style ...string) string
	FullURL(style ...string) string
	Ext() string
	String() string

	RemoveOld() (found bool, err error)
	Old() *Old

	Names() []string
}

// Size is a struct, used for `GetSizes` method, it will return a slice of Size, media library will crop images automatically based on it
type Size struct {
	Width  int
	Height int
}

// URLTemplater is a interface to return url template
type URLTemplater interface {
	GetURLTemplate(*Option) string
}

// Option media library option
type Option map[string]string

// Get used to get option with name
func (option Option) Get(key string) string {
	return option[strings.ToUpper(key)]
}

// Get used to set option with name
func (option *Option) Set(key, value string) *Option {
	(*option)[strings.ToUpper(key)] = value
	return option
}
