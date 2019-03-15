package media

import (
	"database/sql"
	"database/sql/driver"
	"io"
	"os"
	"reflect"
	"strings"

	"github.com/aghape/core/utils"

	"github.com/aghape/core"
	"github.com/aghape/oss"
	"github.com/moisespsena-go/aorm"
)

const (
	FIELD_TAG_NAME = "media"
	OPT_STORAGE    = FIELD_TAG_NAME + ".storage"
)

// Media is an interface including methods that needs for a media library storage
type Media interface {
	IsZero() bool
	HasFile() bool

	sql.Scanner

	Site() core.SiteInterface
	SetSite(site core.SiteInterface)
	Storage() oss.NamedStorageInterface
	SetStorage(storage oss.NamedStorageInterface)
	Init(site core.SiteInterface, field *aorm.Field)
	FieldOption() *Option
	SetFieldOption(option *Option)

	Set(ctx *Context, data interface{}) (err error)

	Value() (driver.Value, error)
	GetURL(scope *aorm.Scope, field *aorm.Field, templater URLTemplater) string

	GetFileHeader() FileHeader
	GetFileName() string

	Store(url string, reader io.Reader) error
	Retrieve(url string) (*os.File, error)
	Remove(url string) (found bool, err error)
	RemoveAll(m Media) (found bool, err error)

	ScanBytes(ctx *Context, data []byte) error

	IsImage() bool

	GetURLTemplate(option *Option) string
	URL(style ...string) string
	FullURL(ctx *core.Context, style ...string) string
	Ext() string
	String() string

	Names() []string
	SystemNames() []string
	AllNames(m Media) []string

	MediaScan(ctx *Context, data interface{}) (err error)

	OlderURL() []string

	Deletable() bool
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

// SetDefault used to set default option with name
func (option *Option) SetDefault(key, value string) *Option {
	if _, ok := (*option)[key]; !ok {
		(*option)[strings.ToUpper(key)] = value
	}
	return option
}

// Merge used to merge options map
func (option *Option) Merge(m ...map[string]string) *Option {
	for _, m := range m {
		for k, v := range m {
			(*option)[k] = v
		}
	}
	return option
}

// MergePrefix used to merge options map with prefix
func (option *Option) MergePrefix(prefix string, m ...map[string]string) *Option {
	for _, m := range m {
		for k, v := range m {
			(*option)[strings.ToUpper(prefix+"."+k)] = v
		}
	}
	return option
}

// GetPrefix used to get options map by prefix
func (option *Option) GetPrefix(prefix string) map[string]string {
	prefix = strings.ToUpper(prefix) + "."
	r := map[string]string{}
	for k, v := range *option {
		if strings.HasPrefix(k, prefix) {
			r[k[len(prefix):]] = v
		}
	}
	return r
}

// ParseTagPrefix used to parse field tag with prefix
func (option *Option) ParseTagPrefix(prefix, tag string) *Option {
	return option.MergePrefix(prefix, utils.ParseTagOption(tag))
}

// ParseFieldTag used to parse field tag with prefix
func (option *Option) ParseFieldTag(tagName string, tag *reflect.StructTag) *Option {
	return option.MergePrefix(tagName, utils.ParseTagOption(tag.Get(tagName)))
}
