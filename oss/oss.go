package oss

import (
	"fmt"
	"io"
	"os"

	"github.com/aghape/media"
	"github.com/aghape/oss"
	"github.com/aghape/oss/filesystem"
	manager "github.com/aghape/oss/manager"
	"github.com/moisespsena-go/aorm"
)

var (
	// Storage the storage used to save medias
	FileSystemStorage = filesystem.New("./data")
	// URLTemplate default URL template
	URLTemplate = "/system/{{class}}/{{primary_key}}/{{column}}/{{filename_with_hash}}"
)

func init() {
	manager.Storages.Default = FileSystemStorage
	manager.Storages.DefaultFS = FileSystemStorage
	aorm.StructFieldMethodCallbacks.RegisterFieldType(&OSS{})
}

const OPTION_KEY = "storage"

func DefaultFilesystemStorage() *filesystem.FileSystem {
	return FileSystemStorage
}

// OSS common storage interface
type OSS struct {
	media.Base
}

// DefaultStoreHandler used to store reader with default Storage
var DefaultStoreHandler = func(storage oss.StorageInterface, path string, option *media.Option, reader io.Reader) error {
	_, err := storage.Put(path, reader)
	return err
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
func (o *OSS) Remove(path string) error {
	return DefaultRemoveHandler(o.Storage(), path, o.FieldOption())
}

func (o *OSS) RemoveOld() (found bool, err error) {
	if o.Old != nil {
		remove := func(url string) (bool, error) {
			_, notFound, err := o.Storage().Stat(url)
			if err != nil {
				return true, fmt.Errorf("Get stat for %q fail: %v", url, err)
			}
			if notFound {
				return false, nil
			}
			err = o.Remove(url)
			if err != nil {
				return true, fmt.Errorf("Remove %q fail: %v", url, err)
			}
			return true, nil
		}
		old := o.Old()
		found, err = remove(old.Url)
		if err != nil {
			return found, err
		}
		if found {
			for _, key := range old.Names() {
				_, err = remove(media.MediaStyleURL(old.Url, key))
				if err != nil {
					return true, err
				}
			}
		}
	}
	return
}

// DefaultRetrieveHandler used to retrieve file
var DefaultRetrieveHandler = func(storage oss.StorageInterface, option *media.Option, path string) (*os.File, error) {
	return storage.Get(path)
}

// Retrieve retrieve file content with url
func (o *OSS) Retrieve(path string) (*os.File, error) {
	return DefaultRetrieveHandler(o.Storage(), o.FieldOption(), path)
}
