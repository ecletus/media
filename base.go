package media

import (
	"bytes"
	"database/sql/driver"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
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

	errwrap "github.com/moisespsena-go/error-wrap"

	"github.com/dustin/go-humanize"
	"github.com/ecletus/admin"
	"github.com/ecletus/core"
	"github.com/ecletus/core/helpers"
	"github.com/ecletus/core/resource"
	"github.com/ecletus/core/utils"
	"github.com/ecletus/oss"
	"github.com/gosimple/slug"
	"github.com/jinzhu/inflection"
	"github.com/moisespsena-go/aorm"
)

var (
	// URLTemplate default URL template
	URLTemplate = "/system/{{class}}/{{primary_key_path}}/{{column}}/{{filename_slug}}"
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
	Delete      bool       `json:"-"`
	FileHeader  FileHeader `json:"-"`
	Reader      io.Reader  `json:"-"`
	site        *core.Site
	storage     oss.NamedStorageInterface
	field       *aorm.Field
	fieldOption *Option
	old         []string
}

func (Base) AormDataType(dialect aorm.Dialector) string {
	switch dialect.GetName() {
	case "postgres":
		return "JSONB"
	}
	return "TEXT"
}

func (b *Base) Store(url string, reader io.Reader) error {
	panic("implement me")
}

func (b *Base) OlderURL() []string {
	return b.old
}

func (b *Base) Names() (names []string) {
	return names
}

func (b *Base) SystemNames() (names []string) {
	return names
}

func (b *Base) Site() *core.Site {
	return b.site
}

func (b *Base) SetSite(site *core.Site) {
	b.site = site
}

func (b *Base) Storage() oss.NamedStorageInterface {
	return b.storage
}

func (b *Base) Remove(url string) (found bool, err error) {
	panic("not implemented error")
}

func (b *Base) RemoveAll(media Media) (found bool, err error) {
	if found, err = media.Remove(media.URL()); err != nil {
		return found, errwrap.Wrap(err, "Remove %q", media.GetFileName())
	}

	for _, key := range append(media.SystemNames(), media.Names()...) {
		var f bool
		if f, err = media.Remove(media.URL(key)); err != nil {
			return f, errwrap.Wrap(err, "Remove style %q", key)
		}
		if f {
			found = true
		}
	}
	return
}

func (b *Base) SetStorage(storage oss.NamedStorageInterface) {
	b.storage = storage
}

func (b *Base) FieldOption() *Option {
	return b.fieldOption
}

func (b *Base) GetOrSetFieldOption() *Option {
	if b.fieldOption == nil {
		option := make(Option)
		b.fieldOption = &option
	}
	return b.fieldOption
}

func (b *Base) SetFieldOption(option *Option) {
	b.fieldOption = option
}

func (b *Base) setFile(fileName string, fileSize int64, fileHeader FileHeader) {
	b.FileName = fileName
	b.FileHeader = fileHeader
	b.FileSize = fileSize
}

func (b *Base) setZero() {
	b.setFile("", 0, nil)
	b.Url = ""
}

func (b *Base) AllNames(m Media) []string {
	return append(m.SystemNames(), m.Names()...)
}

func (b *Base) Set(ctx *Context, data interface{}) (err error) {
	var (
		fileName string
		fileType string
		size     uint64
		check    []func()

		currentFileName = b.FileName
		currentFileSize = b.FileSize

		m = ctx.Media
	)

	if m.HasFile() {
		var currentUrls = []string{b.Url}
		for _, key := range m.AllNames(m) {
			if url := m.URL(key); url != "" {
				currentUrls = append(currentUrls, url)
			}
		}

		defer func() {
			if err == nil {
				if currentFileName != b.FileName || currentFileSize != b.FileSize {
					b.old = currentUrls
				}
			}
		}()
	}

	if t, ok := m.(AcceptTypes); ok {
		ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(fileName), "."))
		check = append(check, func() {
			if fileType != "" {
				for _, typ := range t.FileTypes() {
					if typ == ext {
						return
					}
				}
				err = fmt.Errorf("Invalid file type %q", ext)
			}
		})
	}

	if e, ok := m.(AcceptExts); ok {
		check = append(check, func() {
			ext := strings.ToLower(filepath.Ext(fileName)[1:])
			for _, typ := range e.FileExts() {
				if ext[0] == '.' {
					ext = ext[1:]
				}
				if typ == ext {
					return
				}
			}
			err = fmt.Errorf("Invalid file extension %q", ext)
		})
	}

	if ms, ok := m.(MaxSize); ok {
		check = append(check, func() {
			maxSize := ms.MaxSize()
			if size > maxSize {
				err = fmt.Errorf("Very large file. The expected maximum size is %s, but obtained %s.",
					humanize.Bytes(maxSize), humanize.Bytes(size))
			}
		})
	}

	if check != nil {
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
			fileType = values.Header.Get("Content-Type")
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
						fileType = file.Header.Get("Content-Type")
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

	return m.MediaScan(ctx, data)
}

func (b *Base) ContextScan(ctx *core.Context, data interface{}) (err error) {
	return b.Set(NewContext(b), data)
}

func (b *Base) ScanBytes(ctx *Context, data []byte) (err error) {
	if len(data) == 0 {
		b.setZero()
	} else {
		if err = json.Unmarshal(data, b); err == nil {
			var options struct {
				Delete bool
			}
			if err = json.Unmarshal(data, &options); err == nil {
				if options.Delete {
					b.Delete = true
				}
			}
		}
	}
	return
}

func (b *Base) MediaScan(ctx *Context, data interface{}) (err error) {
	if data == nil {
		b.setZero()
		return nil
	}
	switch values := data.(type) {
	case *os.File:
		stat, err := values.Stat()
		if err != nil {
			return err
		}
		b.setFile(filepath.Base(values.Name()), stat.Size(), &fileWrapper{values})
	case *multipart.FileHeader:
		b.setFile(values.Filename, values.Size, values)
	case []*multipart.FileHeader:
		if len(values) > 0 {
			return b.Scan(values[0])
		}
	case []byte:
		return b.ScanBytes(ctx, values)
	case string:
		return b.ScanBytes(ctx, []byte(values))
	case []string:
		for _, str := range values {
			if err := b.ScanBytes(ctx, []byte(str)); err != nil {
				return err
			}
		}
	default:
		err = errors.New("unsupported driver -> Scan pair for MediaLibrary")
	}
	return
}

func (b *Base) Scan(data interface{}) (err error) {
	return b.MediaScan(NewContext(b), data)
}

func (b Base) DBValue(media Media) (driver.Value, error) {
	if media.IsZero() {
		return nil, nil
	}
	results, err := json.Marshal(media)
	return string(results), err
}

// Value return struct's Value
func (b Base) Value() (driver.Value, error) {
	return b.DBValue(&b)
}

func (b Base) Ext() string {
	return strings.ToLower(path.Ext(b.Url))
}

// URL return file's url with given style
func (b Base) URL(styles ...string) string {
	if b.IsZero() {
		return ""
	}
	if b.Url != "" && len(styles) > 0 && styles[0] != "" {
		return MediaStyleURL(b.Url, styles[0])
	}
	return b.Url
}

// gorm/scope.scan()
func (b *Base) AfterScan(db *aorm.DB, field *aorm.Field) {
	b.Init(core.GetSiteFromDB(db), field)
}

func (b *Base) Init(site *core.Site, field *aorm.Field) {
	if b.site == nil {
		b.site = site
	}

	if field != nil {
		b.GetOrSetFieldOption().ParseFieldTag("media", &field.Tag)
	}

	if b.storage == nil {
		b.storage = b.site.GetMediaStorageOrDefault(b.fieldOption.Get(OPT_STORAGE))
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
		path = URLTemplate
	}
	return
}

func (b Base) FullURL(ctx *core.Context, styles ...string) (url string) {
	if b.IsZero() {
		return ""
	}
	ep := helpers.GetStorageEndpointFromContext(ctx, b.storage)
	url = b.URL(styles...)
	url = ep + url
	url = strings.TrimSuffix(url, "/")
	if strings.HasPrefix(url, "/") {
		return url
	}
	return MediaURL(url)
}

func (b Base) FullURLU(ctx *core.Context, styles ...string) (url string) {
	if b.IsZero() {
		return ""
	}
	return b.FullURL(ctx, styles...) + "?_=" + strconv.Itoa(time.Now().Nanosecond())
}

var urlReplacer = regexp.MustCompile("(\\s|\\+)+")

func getFuncMap(scope *aorm.Scope, field *aorm.Field, filename string) template.FuncMap {
	hash := func() string { return strings.Replace(time.Now().Format("20060102150506.000000000"), ".", "", -1) }
	slugFileName := func() string {
		return slug.Make(strings.TrimSuffix(path.Base(filename), path.Ext(filename)))
	}
	return template.FuncMap{
		"class":       func() string { return inflection.Plural(utils.ToParamString(scope.Struct().Type.Name())) },
		"primary_key": func() string { return fmt.Sprintf("%v", scope.PrimaryKey()) },
		"primary_key_path": func() string {
			var b = base64.RawURLEncoding.EncodeToString(scope.Instance().ID().Bytes())
			var parts = []string{}
			for i := 0; len(b) > (2+i) && len(parts) < 3; i++ {
				parts = append(parts, b[0:2+i])
				b = b[2+i:]
			}
			if len(b) > 0 {
				parts = append(parts, b)
			}
			return strings.Join(parts, "/")
		},
		"column":   func() string { return strings.ToLower(field.Name) },
		"filename": func() string { return filename },
		"filename_slug": func() string {
			return urlReplacer.ReplaceAllString(slugFileName()+path.Ext(filename), "-")
		},
		"basename": func() string { return strings.TrimSuffix(path.Base(filename), path.Ext(filename)) },
		"hash":     hash,
		"filename_with_hash": func() string {
			return urlReplacer.ReplaceAllString(fmt.Sprintf("%s.%v%v", slugFileName(), hash(), path.Ext(filename)), "-")
		},
		"extension": func() string { return strings.TrimPrefix(path.Ext(filename), ".") },
	}
}

// GetURL get default URL for a model based on its options
func (b Base) GetURL(scope *aorm.Scope, field *aorm.Field, templater URLTemplater) string {
	if pth := templater.GetURLTemplate(b.fieldOption); pth != "" {
		tmpl := template.New("").Funcs(getFuncMap(scope, field, b.GetFileName()))
		if tmpl, err := tmpl.Parse(pth); err == nil {
			var result = bytes.NewBufferString("")
			if err := tmpl.Execute(result, scope.Value); err == nil {
				return result.String()
			}
		}
	}
	return ""
}

// Retrieve retrieve file content with url
func (b Base) Retrieve(url string) (*os.File, error) {
	return nil, errors.New("not implemented")
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

func (b Base) HasFile() bool {
	return b.FileName != ""
}

func (b Base) IsZero() bool {
	return !b.HasFile()
}

func (b Base) Deletable() bool {
	return b.Delete
}

// ConfigureQorMetaBeforeInitialize configure this field for Qor Admin
func (Base) ConfigureQorMetaBeforeInitialize(meta resource.Metaor) {
	if meta, ok := meta.(*admin.Meta); ok {
		if meta.Type == "" {
			meta.Type = "media_file"
		}

		if meta.GetFormattedValuer() == nil {
			meta.SetFormattedValuer(func(value interface{}, context *core.Context) interface{} {
				return utils.StringifyContext(meta.Value(context, value), context)
			})
		}
	}
}
