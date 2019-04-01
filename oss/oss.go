package oss

import (
	"fmt"
	"io"
	"os"

	"github.com/ecletus/core"

	"github.com/ecletus/media"
	"github.com/ecletus/oss"
	"github.com/ecletus/oss/filesystem"
	manager "github.com/ecletus/oss/manager"
	"github.com/moisespsena-go/aorm"
	"github.com/moisespsena-go/path-helpers"
)

var (
	// Storage the storage used to save medias
	FileSystemStorage = filesystem.New(&filesystem.Config{RootDir: "./data"})
	PKG               = path_helpers.GetCalledDir()
)

func init() {
	manager.Storages.Default = FileSystemStorage
	manager.Storages.DefaultFS = FileSystemStorage
	aorm.StructFieldMethodCallbacks.RegisterFieldType(&OSS{})
}

type OSSInterface interface {
	media.Media
	IsNew() bool
}

// OSS common storage interface
type OSS struct {
	media.Base
	isNew   bool
	callSet bool
}

func (o OSS) IsNew() bool {
	return o.isNew
}

// DefaultStoreHandler used to store reader with default Storage
var DefaultStoreHandler = func(storage oss.StorageInterface, path string, option *media.Option, reader io.Reader) error {
	_, err := storage.Put(path, reader)
	return err
}

func (o *OSS) Init(site core.SiteInterface, field *aorm.Field) {
	o.Base.Init(site, field)
	o.GetOrSetFieldOption().ParseFieldTag("oss", &field.Tag)
}

// Store save reader's content with path
func (o *OSS) Store(path string, reader io.Reader) error {
	return DefaultStoreHandler(o.Storage(), path, o.FieldOption(), reader)
}

// DefaultRemoveHandler used to store reader with default Storage
var DefaultRemoveHandler = func(storage oss.StorageInterface, path string, option *media.Option) error {
	return storage.Delete(path)
}

// Remove content by path
func (o *OSS) Remove(path string) (found bool, err error) {
	_, notFound, err := o.Storage().Stat(path)
	if err != nil {
		return true, fmt.Errorf("Get stat for %q fail: %v", path, err)
	}
	if notFound {
		return false, nil
	}
	err = DefaultRemoveHandler(o.Storage(), path, o.FieldOption())
	if err != nil {
		return true, fmt.Errorf("Remove %q fail: %v", path, err)
	}
	return true, nil
}

func (o *OSS) Set(ctx *media.Context, data interface{}) (err error) {
	if !o.callSet {
		o.callSet = true
		defer func() {
			o.callSet = false
		}()
	}
	header := o.FileHeader
	if err = o.Base.Set(ctx, data); err != nil {
		return
	}
	if !o.isNew && header == nil && o.FileHeader != nil {
		o.isNew = true
	}
	return
}

func (o *OSS) Scan(data interface{}) (err error) {
	return o.MediaScan(media.NewContext(o), data)
}

func (o *OSS) ContextScan(ctx *core.Context, data interface{}) (err error) {
	return o.Set(media.NewContext(o), data)
}

func (o *OSS) MediaScan(ctx *media.Context, data interface{}) (err error) {
	return o.Base.MediaScan(ctx, data)
}

// DefaultRetrieveHandler used to retrieve file
var DefaultRetrieveHandler = func(storage oss.StorageInterface, option *media.Option, path string) (*os.File, error) {
	return storage.Get(path)
}

// Retrieve retrieve file content with url
func (o *OSS) Retrieve(path string) (*os.File, error) {
	return DefaultRetrieveHandler(o.Storage(), o.FieldOption(), path)
}
