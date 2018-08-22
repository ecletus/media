package media

import (
	"encoding/json"
	"errors"
	"mime/multipart"
	"reflect"

	"github.com/moisespsena-go/aorm"
	"github.com/aghape/core"
	"github.com/aghape/serializable_meta"
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
func fieldOption(field *aorm.Field) *Option {
	return parseTagOption(field.Tag.Get("media_library"))
}

func cropField(field *aorm.Field, scope *aorm.Scope) (cropped bool) {
	if field.Field.CanAddr() {
		// TODO Handle scanner
		if media, ok := field.Field.Addr().Interface().(Media); ok && !media.Cropped() {
			media.Init(core.GetSiteFromDB(scope.DB()), field)
			option := media.FieldOption()

			if media.GetFileHeader() != nil || media.NeedCrop() {
				var file multipart.File
				var err error
				if fileHeader := media.GetFileHeader(); fileHeader != nil {
					file, err = media.GetFileHeader().Open()
				} else {
					file, err = media.Retrieve(media.URL("original"))
				}

				if err != nil {
					scope.Err(err)
					return false
				}

				media.Cropped(true)

				if url := media.GetURL(scope, field, media); url == "" {
					scope.Err(errors.New("invalid URL"))
				} else {
					result, _ := json.Marshal(map[string]string{"Url": url})
					media.Scan(string(result))
				}

				if file != nil {
					defer file.Close()
					var handled = false
					for _, handler := range mediaHandlers {
						if handler.CouldHandle(media) {
							file.Seek(0, 0)
							if scope.Err(handler.Handle(media, file, option)) == nil {
								handled = true
							}
						}
					}

					// Save File
					if !handled {
						scope.Err(media.Store(media.URL(), file))
					}
				}
				return true
			}
		}
	}
	return false
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
					isCropped        bool
					handleNestedCrop func(record interface{})
				)

				handleNestedCrop = func(record interface{}) {
					newScope := scope.New(record)
					for _, field := range newScope.Fields() {
						if cropField(field, scope) {
							isCropped = true
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
				if isCreate && isCropped {
					updateColumns["value"], _ = json.Marshal(record)
				}
			}

			// Handle Normal Field
			for _, field := range scope.Fields() {
				if cropField(field, scope) && isCreate {
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
