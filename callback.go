package media

import (
	"reflect"

	"github.com/moisespsena-go/error-wrap"

	"github.com/ecletus/core"
	"github.com/moisespsena-go/aorm"
)

var E_DELETE = PKG + ":delete"
var DB_DELETE_IGNORE = PKG + ".delete.ignore"

func IgnoreCallback(db *aorm.DB) *aorm.DB {
	return db.Set(DB_DELETE_IGNORE, true)
}

func IsIgnoreCallback(v interface{}) bool {
	switch t := v.(type) {
	case *aorm.DB:
		v, ok := t.Get(DB_DELETE_IGNORE)
		return v != nil && ok
	case *aorm.Scope:
		v, ok := t.Get(DB_DELETE_IGNORE)
		return v != nil && ok
	}
	return false
}

func deleteField(field *aorm.Field, scope *aorm.Scope) (changed bool) {
	if field.Field.CanAddr() {
		if media, ok := field.Field.Addr().Interface().(Media); ok {
			media.Init(core.GetSiteFromDB(scope.DB()), field)

			for _, url := range media.OlderURL() {
				if _, err := media.Remove(url); err != nil {
					scope.Err(errwrap.Wrap(err, "Remove OLD media %q", url))
					return false
				}
			}

			if media.Deletable() {
				_, err := media.RemoveAll(media)
				if err != nil {
					scope.Err(errwrap.Wrap(err, "Remove media"))
				} else {
					field.Field.Set(reflect.New(field.Struct.Type).Elem())
				}
				return true
			}

			return false
		}
	}
	return changed
}

func deleteCallback(scope *aorm.Scope) {
	if IsIgnoreCallback(scope) {
		return
	}
	if !scope.HasError() {
		// TODO: Handle update attrs
		// "gorm:update_attrs"
		// Handle Normal Field
		for _, field := range scope.Fields() {
			deleteField(field, scope)
		}
	}
}

// RegisterCallbacks register callback into GORM DB
func RegisterCallbacks(db *aorm.DB) {
	db.Callback().Update().Before("gorm:before_update").Register(E_DELETE, deleteCallback)
	db.Callback().Delete().Before("gorm:before_delete").Register(E_DELETE, deleteCallback)
}
