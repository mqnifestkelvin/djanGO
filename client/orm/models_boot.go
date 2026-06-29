// Copyright 2014 beego Author. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package orm

import (
	"fmt"
	"runtime/debug"

	imodels "github.com/mqnifestkelvin/djanGO/client/orm/internal/models"
)

var defaultModelCache = imodels.NewModelCacheHandler()

// RegisterModel Register models
func RegisterModel(models ...interface{}) {
	RegisterModelWithPrefix("", models...)
}

// RegisterModelWithPrefix Register models with a prefix
func RegisterModelWithPrefix(prefix string, models ...interface{}) {
	if err := defaultModelCache.Register(prefix, true, models...); err != nil {
		panic(err)
	}
}

// RegisterModelWithSuffix Register models with a suffix
func RegisterModelWithSuffix(suffix string, models ...interface{}) {
	if err := defaultModelCache.Register(suffix, false, models...); err != nil {
		panic(err)
	}
}

// BootStrap Bootstrap models.
// make All model parsed and can not add more models
func BootStrap() {
	BootStrapWithAlias("default")
}

// BootStrap with alias
func BootStrapWithAlias(alias string) {
	if _, ok := dataBaseCache.get(alias); !ok {
		fmt.Printf("must have one Register DataBase alias named %q\n", alias)
		debug.PrintStack()
		return
	}
	defaultModelCache.Bootstrap()
}

// ResetModelCache Clean model cache. Then you can re-RegisterModel.
// Common use this api for test case.
func ResetModelCache() {
	defaultModelCache.Clean()
}

// ModelFieldInfo is a public representation of a registered model field,
// used by core/migration for schema introspection (makemigrations).
type ModelFieldInfo struct {
	Name       string
	ColumnName string
	FieldType  int // matches imodels.Type* constants, re-exported below as ORM* consts
	Size       int
	Null       bool
	Unique     bool
	PrimaryKey bool
	Auto       bool
	Index      bool
	Digits     int
	Decimals   int
	RelTable   string
	Default    string // string representation of the default value, empty = none
}

// ModelInfo is a public representation of a registered model,
// used by core/migration for schema introspection (makemigrations).
type ModelInfo struct {
	Name     string
	FullName string // e.g. "blog.Post"
	Table    string
	Fields   []ModelFieldInfo
}

// Field type constants re-exported so core/migration can compare without
// importing the internal package.
const (
	ORMTypeBooleanField              = imodels.TypeBooleanField
	ORMTypeVarCharField              = imodels.TypeVarCharField
	ORMTypeCharField                 = imodels.TypeCharField
	ORMTypeTextField                 = imodels.TypeTextField
	ORMTypeTimeField                 = imodels.TypeTimeField
	ORMTypeDateField                 = imodels.TypeDateField
	ORMTypeDateTimeField             = imodels.TypeDateTimeField
	ORMTypeSmallIntegerField         = imodels.TypeSmallIntegerField
	ORMTypeIntegerField              = imodels.TypeIntegerField
	ORMTypeBigIntegerField           = imodels.TypeBigIntegerField
	ORMTypePositiveSmallIntegerField = imodels.TypePositiveSmallIntegerField
	ORMTypePositiveIntegerField      = imodels.TypePositiveIntegerField
	ORMTypePositiveBigIntegerField   = imodels.TypePositiveBigIntegerField
	ORMTypeFloatField                = imodels.TypeFloatField
	ORMTypeDecimalField              = imodels.TypeDecimalField
	ORMTypeJSONField                 = imodels.TypeJSONField
	ORMTypeJsonbField                = imodels.TypeJsonbField
	ORMRelForeignKey                 = imodels.RelForeignKey
	ORMRelOneToOne                   = imodels.RelOneToOne
	ORMRelManyToMany                 = imodels.RelManyToMany
)

// GetRegisteredModels returns all registered models as public ModelInfo structs
// for inspection by the migration system. Call after RegisterModel().
func GetRegisteredModels() []ModelInfo {
	var out []ModelInfo
	for _, mi := range defaultModelCache.AllOrdered() {
		info := ModelInfo{
			Name:     mi.Name,
			FullName: mi.FullName,
			Table:    mi.Table,
		}
		for _, fi := range mi.Fields.FieldsDB {
			mfi := ModelFieldInfo{
				Name:       fi.Name,
				ColumnName: fi.Column,
				FieldType:  fi.FieldType,
				Size:       fi.Size,
				Null:       fi.Null,
				Unique:     fi.Unique,
				PrimaryKey: fi.Pk,
				Auto:       fi.Auto,
				Index:      fi.Index,
				Digits:     fi.Digits,
				Decimals:   fi.Decimals,
				RelTable:   fi.RelTable,
			}
			if fi.Initial.Exist() {
				mfi.Default = "'" + fi.Initial.String() + "'"
			}
			info.Fields = append(info.Fields, mfi)
		}
		out = append(out, info)
	}
	return out
}
