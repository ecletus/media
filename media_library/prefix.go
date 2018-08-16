package media_library

import (
	"github.com/moisespsena/go-i18n-modular/i18nmod"
	"github.com/moisespsena/go-path-helpers"
)

var (
	PKG       = path_helpers.GetCalledDir()
	I18NGROUP = i18nmod.PkgToGroup(PKG)
)
