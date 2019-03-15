package oss

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/aghape/media"
	"mime/multipart"
	"reflect"

	"github.com/dsnet/golib/memfile"

	"github.com/aghape/core"

	"github.com/aghape/serializable_meta"
	"github.com/moisespsena-go/aorm"
)

var E_SAVE_AND_CROP = PKG + ":save_and_crop"
var DB_CALLBACK_IGNORE = PKG + ".callback.ignore"

func IgnoreCallback(db *aorm.DB) *aorm.DB {
	return db.Set(DB_CALLBACK_IGNORE, true)
}

func IsIgnoreCallback(v interface{}) bool {
	switch t := v.(type) {
	case *aorm.DB:
		v, ok := t.Get(DB_CALLBACK_IGNORE)
		return v != nil && ok
	case *aorm.Scope:
		v, ok := t.Get(DB_CALLBACK_IGNORE)
		return v != nil && ok
	}
	return false
}

func cropField(img ImageInterface) (cropped bool, err error) {
	var file multipart.File
	if fileHeader := img.GetFileHeader(); fileHeader != nil {
		file, err = img.GetFileHeader().Open()
	} else {
		file, err = img.Retrieve(img.URL())
	}

	if err != nil {
		return false, err
	}

	img.Cropped(true)

	if file != nil {
		var cropper *ImageCropper
		if cropper, err = NewImageCropper(img, file); err != nil {
			return false, err
		}

		cb := func(key string, f *bytes.Buffer) (err error) {
			return img.Store(img.URL(key), f)
		}
		var original bool
		if err = cropper.CropNames(func(key string, f *bytes.Buffer) error {
			original = true
			if cropper, err = NewImageCropper(img, memfile.New(f.Bytes())); err != nil {
				return err
			}
			return cb(key, f)
		}, IMAGE_STYLE_ORIGNAL); err != nil {
			return false, err
		}

		size := img.GetOriginalSize()
		size.Width = cropper.Width()
		size.Height = cropper.Height()

		if !original {
			img.Store(img.URL(IMAGE_STYLE_ORIGNAL), file)
			file.Seek(0, 0)
		}

		var names []string
		for _, n := range img.SystemNames() {
			if n != IMAGE_STYLE_ORIGNAL {
				names = append(names, n)
			}
		}

		if img.Cropable() {
			names = append(names, img.Names()...)
		}

		if err = cropper.CropNames(cb, names...); err != nil {
			return false, err
		}
	}
	return true, err
}

func saveField(field *aorm.Field, scope *aorm.Scope) (changed bool) {
	if field.Field.CanAddr() {
		if oss, ok := field.Field.Addr().Interface().(OSSInterface); ok {
			if oss.Deletable() {
				return false
			}
			if !oss.HasFile() {
				return true
			}

			oss.Init(core.GetSiteFromDB(scope.DB()), field)

			var url string

			if oss.IsNew() {
				if url = oss.GetURL(scope, field, oss); url == "" {
					scope.Err(errors.New("invalid URL"))
					return false
				}
				result, _ := json.Marshal(map[string]string{"Url": url})
				oss.MediaScan(media.NewContext(oss, map[interface{}]interface{}{"oss.db_callback":true}), result)
				// is new
				file, err := oss.GetFileHeader().Open()
				if err != nil {
					scope.Err(err)
					return false
				}
				defer file.Close()
				if err = oss.Store(url, file); err != nil {
					scope.Err(err)
					return false
				}
			}

			if img, ok := oss.(ImageInterface); ok {
				if img.Cropped() {
					return false
				}
				if img.IsNew() || img.NeedCrop() {
					var err error
					if changed, err = cropField(img); err != nil {
						scope.Err(err)
					}
				}
			}
		}
	}
	return changed
}

func saveAndCropImage(isCreate bool) func(scope *aorm.Scope) {
	return func(scope *aorm.Scope) {
		if IsIgnoreCallback(scope) {
			return
		}
		if !scope.HasError() {
			var updateColumns = map[string]interface{}{}

			// Handle SerializableMeta
			if value, ok := scope.Value.(serializable_meta.SerializableMetaInterface); ok {
				var (
					changed          bool
					handleNestedCrop func(record interface{})
				)

				handleNestedCrop = func(record interface{}) {
					newScope := scope.New(record)
					for _, field := range newScope.Fields() {
						if changed = saveField(field, scope); changed {
							continue
						}

						if reflect.Indirect(field.Field).Kind() == reflect.Struct {
							handleNestedCrop(field.Field.Addr().Interface())
						}

						if reflect.Indirect(field.Field).Kind() == reflect.Slice {
							for i := 0; i < reflect.Indirect(field.Field).Len(); i++ {
								handleNestedCrop(reflect.Indirect(field.Field).Index(i).Addr().Interface())
							}
						}
					}
				}

				record := value.GetSerializableArgument(value)
				handleNestedCrop(record)
				if isCreate && changed {
					updateColumns["value"], _ = json.Marshal(record)
				}
			}

			// Handle Normal Field
			for _, field := range scope.Fields() {
				if saveField(field, scope) && isCreate {
					updateColumns[field.DBName] = field.Field.Interface()
				}
			}

			if !scope.HasError() && len(updateColumns) != 0 {
				scope.Err(scope.NewDB().Model(scope.Value).UpdateColumns(updateColumns).Error)
			}
		}
	}
}

// RegisterCallbacks register callback into GORM DB
func RegisterCallbacks(db *aorm.DB) {
	db.Callback().Update().Before("gorm:before_update").Register(E_SAVE_AND_CROP, saveAndCropImage(false))
	db.Callback().Create().After("gorm:after_create").Register(E_SAVE_AND_CROP, saveAndCropImage(true))
}
