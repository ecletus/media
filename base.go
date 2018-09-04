package media

import (
	"bytes"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"io"
	"mime/multipart"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/aghape/admin"
	"github.com/aghape/core"
	"github.com/aghape/core/resource"
	"github.com/aghape/core/utils"
	"github.com/aghape/oss"
	"github.com/dustin/go-humanize"
	"github.com/gosimple/slug"
	"github.com/jinzhu/inflection"
	"github.com/moisespsena-go/aorm"
)

type FullURL interface {
	FullURL() string
}
type FullURLStyle interface {
	FullURL(styles ...string) string
}

type FullURLContext interface {
	FullURL(ctx *core.Context) string
}

type FullURLContextStyle interface {
	FullURL(ctx *core.Context, styles ...string) string
}

type URL interface {
	URL() string
}

type URLStyle interface {
	URL(styles ...string) string
}

type URLContext interface {
	URL(ctx *core.Context) string
}

type URLContextStyle interface {
	URL(ctx *core.Context, styles ...string) string
}

// CropOption includes crop options
type CropOption struct {
	X, Y, Width, Height int
}

// FileHeader is an interface, for matched values, when call its `Open` method will return `multipart.File`
type FileHeader interface {
	Open() (multipart.File, error)
}

type fileWrapper struct {
	*os.File
}

func (fileWrapper *fileWrapper) Open() (multipart.File, error) {
	return fileWrapper.File, nil
}

// Base defined a base struct for storages
type Base struct {
	FileName    string
	Url         string
	FileSize    int64
	CropOptions map[string]*CropOption `json:",omitempty"`
	Delete      bool                   `json:"-"`
	Crop        bool                   `json:"-"`
	FileHeader  FileHeader             `json:"-"`
	Reader      io.Reader              `json:"-"`
	cropped     bool
	site        core.SiteInterface
	storage     oss.StorageInterface
	field       *aorm.Field
	fieldOption *Option
	old         *Old `json:"-"`
}

func (b *Base) Old() *Old {
	return b.old
}

func (b *Base) Names() (names []string) {
	return names
}

func (b *Base) Site() core.SiteInterface {
	return b.site
}

func (b *Base) SetSite(site core.SiteInterface) {
	b.site = site
}

func (b *Base) Storage() oss.StorageInterface {
	return b.storage
}

func (b *Base) SetStorage(storage oss.StorageInterface) {
	b.storage = storage
}

func (b *Base) FieldOption() *Option {
	return b.fieldOption
}

func (b *Base) SetFieldOption(option *Option) {
	b.fieldOption = option
}

func (b *Base) SetFile(fileName string, fileSize int64, fileHeader FileHeader) {
	if b.FileName != "" {
		b.old = &Old{b.Url, make(map[string]int)}
		b.old.AddName(b.Names()...)
	}
	b.FileName = fileName
	b.FileHeader = fileHeader
	b.FileSize = fileSize
	b.CropOptions = make(map[string]*CropOption)
}

func (b *Base) CallContextScan(ctx *core.Context, data interface{}, scan func(data interface{}) error) (err error) {
	var (
		types    []string
		fileName string
		size     uint64
		maxSize  uint64
		check    []func()
	)

	if t, ok := data.(AcceptTypes); ok {
		ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(fileName), "."))
		check = append(check, func() {
			for _, typ := range t.FileTypes() {
				if typ == ext {
					return
				}
			}
			err = fmt.Errorf("Invalid file type %q", ext)
		})
	}

	if e, ok := data.(AcceptExts); ok {
		ext := strings.ToLower(filepath.Ext(fileName)[1:])
		check = append(check, func() {
			for _, typ := range e.FileExts() {
				if typ == ext {
					return
				}
			}
			err = fmt.Errorf("Invalid file extension %q", ext)
		})
	}

	if ms, ok := data.(MaxSize); ok {
		check = append(check, func() {
			maxSize := ms.MaxSize()
			if size > maxSize {
				err = fmt.Errorf("Very large file. The expected maximum size is %s, but obtained %s.",
					humanize.Bytes(maxSize), humanize.Bytes(size))
			}
		})
	}

	if len(types) > 0 || maxSize > 0 {
		switch values := data.(type) {
		case *os.File:
			fileName = values.Name()
			var stat os.FileInfo
			if stat, err = values.Stat(); err != nil {
				return
			}
			size = uint64(stat.Size())
		case *multipart.FileHeader:
			fileName = values.Filename
			size = uint64(values.Size)
		case []*multipart.FileHeader:
			if len(values) == 1 {
				if file := values[0]; file.Size > 0 {
					fileName = file.Filename
					size = uint64(file.Size)
				}
			} else if len(values) > 1 {
				for i, file := range values {
					if file.Size > 0 {
						fileName = file.Filename
						size = uint64(file.Size)
						for _, cb := range check {
							cb()
							if err != nil {
								return fmt.Errorf("File #%d: %v", i, err)
							}
						}
					}
				}
				check = []func(){}
			}
		}

		if fileName != "" && size > 0 {
			for _, cb := range check {
				cb()
				if err != nil {
					return err
				}
			}
		}
	}

	return scan(data)
}

func (b *Base) ContextScan(ctx *core.Context, data interface{}) (err error) {
	return b.CallContextScan(ctx, data, b.Scan)
}

// Scan scan files, crop options, db values into struct
func (b *Base) Scan(data interface{}) (err error) {
	if data == nil {
		b.SetFile("", 0, &fileWrapper{})
		return nil
	}
	switch values := data.(type) {
	case *os.File:
		stat, err := values.Stat()
		if err != nil {
			return err
		}
		b.SetFile(filepath.Base(values.Name()), stat.Size(), &fileWrapper{values})
	case *multipart.FileHeader:
		b.SetFile(values.Filename, values.Size, values)
	case []*multipart.FileHeader:
		if len(values) > 0 {
			return b.Scan(values[0])
		}
	case []byte:
		if string(values) != "" {
			if err = json.Unmarshal(values, b); err == nil {
				var options struct {
					Crop   bool
					Delete bool
				}
				if err = json.Unmarshal(values, &options); err == nil {
					if options.Crop {
						b.Crop = true
					}
					if options.Delete {
						b.Delete = true
					}
				}
			}
		}
	case string:
		return b.Scan([]byte(values))
	case []string:
		for _, str := range values {
			if err := b.Scan(str); err != nil {
				return err
			}
		}
	default:
		err = errors.New("unsupported driver -> Scan pair for MediaLibrary")
	}

	// If image is deleted, then clean up all values, for serialized fields
	if b.Delete {
		b.Url = ""
		b.FileName = ""
		b.CropOptions = nil
	}
	return
}

// Value return struct's Value
func (b Base) Value() (driver.Value, error) {
	if b.Delete {
		return nil, nil
	}

	results, err := json.Marshal(b)
	return string(results), err
}

func (b Base) Ext() string {
	return strings.ToLower(path.Ext(b.Url))
}

// URL return file's url with given style
func (b Base) URL(styles ...string) string {
	if b.Url != "" && len(styles) > 0 && styles[0] != "" {
		return MediaStyleURL(b.Url, styles[0])
	}
	return b.Url
}

// gorm/scope.scan()
func (b *Base) AfterScan(db *aorm.DB, field *aorm.Field) {
	b.Init(core.GetSiteFromDB(db), field)
}

func (b *Base) Init(site core.SiteInterface, field *aorm.Field) {
	if b.site == nil {
		b.site = site
	}
	if b.fieldOption == nil {
		b.fieldOption = fieldOption(field)
	}
	if b.storage == nil {
		b.storage = b.site.GetMediaStorageOrDefault(b.fieldOption.Get("storage"))
	}
}

// String return file's url
func (b Base) String() string {
	return b.URL()
}

// GetFileName get file's name
func (b Base) GetFileName() string {
	return b.FileName
}

// GetFileHeader get file's header, this value only exists when saving files
func (b Base) GetFileHeader() FileHeader {
	return b.FileHeader
}

// GetURLTemplate get url template
func (b Base) GetURLTemplate(option *Option) (path string) {
	if path = b.fieldOption.Get("URL"); path == "" {
		path = "/system/{{class}}/{{primary_key}}/{{column}}/{{filename_with_hash}}"
	}
	return
}

func (b Base) FullURL(styles ...string) (url string) {
	url = strings.TrimSuffix(strings.Join([]string{b.storage.GetEndpoint(),
		strings.TrimPrefix(b.URL(styles...), "/")}, "/"), "/")
	if strings.HasPrefix(url, "/") {
		return url
	}
	return MediaURL(url)
}

func (b Base) FullURLU(styles ...string) (url string) {
	return b.FullURL(styles...) + "?_=" + strconv.Itoa(time.Now().Nanosecond())
}

var urlReplacer = regexp.MustCompile("(\\s|\\+)+")

func getFuncMap(scope *aorm.Scope, field *aorm.Field, filename string) template.FuncMap {
	hash := func() string { return strings.Replace(time.Now().Format("20060102150506.000000000"), ".", "", -1) }
	return template.FuncMap{
		"class":       func() string { return inflection.Plural(utils.ToParamString(scope.GetModelStruct().ModelType.Name())) },
		"primary_key": func() string { return fmt.Sprintf("%v", scope.PrimaryKeyValue()) },
		"column":      func() string { return strings.ToLower(field.Name) },
		"filename":    func() string { return filename },
		"basename":    func() string { return strings.TrimSuffix(path.Base(filename), path.Ext(filename)) },
		"hash":        hash,
		"filename_with_hash": func() string {
			return urlReplacer.ReplaceAllString(fmt.Sprintf("%s.%v%v", slug.Make(strings.TrimSuffix(path.Base(filename), path.Ext(filename))), hash(), path.Ext(filename)), "-")
		},
		"extension": func() string { return strings.TrimPrefix(path.Ext(filename), ".") },
	}
}

// GetURL get default URL for a model based on its options
func (b Base) GetURL(scope *aorm.Scope, field *aorm.Field, templater URLTemplater) string {
	if path := templater.GetURLTemplate(b.fieldOption); path != "" {
		tmpl := template.New("").Funcs(getFuncMap(scope, field, b.GetFileName()))
		if tmpl, err := tmpl.Parse(path); err == nil {
			var result = bytes.NewBufferString("")
			if err := tmpl.Execute(result, scope.Value); err == nil {
				return result.String()
			}
		}
	}
	return ""
}

// Cropped mark the image to be cropped
func (b *Base) Cropped(values ...bool) (result bool) {
	result = b.cropped
	for _, value := range values {
		b.cropped = value
	}
	return result
}

// NeedCrop return the file needs to be cropped or not
func (b *Base) NeedCrop() bool {
	return b.Crop
}

// GetCropOption get crop options
func (b *Base) GetCropOption(name string) *image.Rectangle {
	if cropOption := b.CropOptions[strings.Split(name, "@")[0]]; cropOption != nil {
		return &image.Rectangle{
			Min: image.Point{X: cropOption.X, Y: cropOption.Y},
			Max: image.Point{X: cropOption.X + cropOption.Width, Y: cropOption.Y + cropOption.Height},
		}
	}
	return nil
}

// Retrieve retrieve file content with url
func (b Base) Retrieve(url string) (*os.File, error) {
	return nil, errors.New("not implemented")
}

// GetSizes get configured sizes, it will be used to crop images accordingly
func (b Base) GetSizes() map[string]*Size {
	return map[string]*Size{}
}

// IsImage return if it is an image
func (b Base) IsImage() bool {
	return IsImageFormat(b.URL())
}

func (b Base) IsVideo() bool {
	return IsVideoFormat(b.URL())
}

func (b Base) IsSVG() bool {
	return IsSVGFormat(b.URL())
}

func (b Base) IsZero() bool {
	return b.FileName == ""
}

// ConfigureQorMetaBeforeInitialize configure this field for Qor Admin
func (Base) ConfigureQorMetaBeforeInitialize(meta resource.Metaor) {
	if meta, ok := meta.(*admin.Meta); ok {
		if meta.Type == "" {
			meta.Type = "file"
		}

		if meta.GetFormattedValuer() == nil {
			meta.SetFormattedValuer(func(value interface{}, context *core.Context) interface{} {
				return utils.StringifyContext(meta.Value(context, value), context)
			})
		}
	}
}
