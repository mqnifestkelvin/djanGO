package migration

import (
	"sort"
	"strings"

	"github.com/mqnifestkelvin/djanGO/client/orm"
)

// Autodetector inspects the current ORM model registry and produces a list of
// Operations needed to bring the database schema up to date.
// Equivalent to Django's MigrationAutodetector in django/db/migrations/autodetector.py.
//
// Django compares from-state (applied migrations) to to-state (current models).
// We simplify to: compare registered models against what migration files already declare.
// For the initial migration this means: all registered models → CreateModel operations.
type Autodetector struct{}

// NewAutodetector creates an Autodetector.
func NewAutodetector() *Autodetector {
	return &Autodetector{}
}

// Changes computes what operations are needed for the given app.
// existingModels is the set of model table names that already have migrations written.
// Returns nil if there are no changes.
//
// Mirrors Django's MigrationAutodetector.changes() — comparing project state
// (registered models) against applied state (existing migration files).
func (a *Autodetector) Changes(app string, existingModels map[string]bool) ([]Operation, error) {
	models := orm.GetRegisteredModels()

	var ops []Operation
	for _, mi := range models {
		if !modelBelongsToApp(mi, app) {
			continue
		}
		if existingModels[mi.Table] {
			continue
		}

		fields := convertFields(mi.Fields)
		ops = append(ops, &CreateModel{
			App:    app,
			Name:   mi.Name,
			Table:  mi.Table,
			Fields: fields,
		})
	}

	sort.Slice(ops, func(i, j int) bool {
		return ops[i].ModelName() < ops[j].ModelName()
	})
	return ops, nil
}

// modelBelongsToApp returns true if the model's package path ends with the app label.
// Beego stores FullName as "import/path.ModelName" e.g. "mysite/blog.Post".
func modelBelongsToApp(mi orm.ModelInfo, app string) bool {
	dotIdx := strings.LastIndex(mi.FullName, ".")
	if dotIdx < 0 {
		return false
	}
	pkgPath := strings.ToLower(mi.FullName[:dotIdx]) // e.g. "mysite/blog"
	appLower := strings.ToLower(app)

	// Match if the last path segment of the package equals the app label
	slashIdx := strings.LastIndex(pkgPath, "/")
	lastSegment := pkgPath
	if slashIdx >= 0 {
		lastSegment = pkgPath[slashIdx+1:]
	}
	return lastSegment == appLower
}

// convertFields maps orm.ModelFieldInfo → migration.FieldDef.
func convertFields(fields []orm.ModelFieldInfo) []FieldDef {
	out := make([]FieldDef, 0, len(fields))
	for _, f := range fields {
		typeName := ormFieldTypeName(f.FieldType)

		// Auto-increment primary key → AutoField (mirrors Django's AutoField)
		if f.Auto && f.PrimaryKey {
			typeName = "AutoField"
		}

		// Strip the ORM-level quote wrapping from bool defaults.
		// Beego stores bool default as "'false'" but we want plain "false".
		defaultVal := f.Default
		if typeName == "BooleanField" && len(defaultVal) > 2 &&
			defaultVal[0] == '\'' && defaultVal[len(defaultVal)-1] == '\'' {
			defaultVal = defaultVal[1 : len(defaultVal)-1]
		}

		out = append(out, FieldDef{
			Name:       f.Name,
			ColumnName: f.ColumnName,
			Type:       typeName,
			MaxSize:    f.Size,
			Null:       f.Null,
			Unique:     f.Unique,
			PrimaryKey: f.PrimaryKey,
			AutoIncr:   f.Auto,
			Index:      f.Index,
			Digits:     f.Digits,
			Decimals:   f.Decimals,
			RelTable:   f.RelTable,
			Default:    defaultVal,
		})
	}
	return out
}

// ormFieldTypeName maps ORM field type integer constants to djanGO type name strings.
// These constants are re-exported from orm package to avoid importing internal/models.
func ormFieldTypeName(ft int) string {
	switch ft {
	case orm.ORMTypeBooleanField:
		return "BooleanField"
	case orm.ORMTypeVarCharField, orm.ORMTypeCharField:
		return "CharField"
	case orm.ORMTypeTextField:
		return "TextField"
	case orm.ORMTypeTimeField:
		return "TimeField"
	case orm.ORMTypeDateField:
		return "DateField"
	case orm.ORMTypeDateTimeField:
		return "DateTimeField"
	case orm.ORMTypeSmallIntegerField:
		return "SmallIntegerField"
	case orm.ORMTypeIntegerField:
		return "IntegerField"
	case orm.ORMTypeBigIntegerField:
		return "BigIntegerField"
	case orm.ORMTypePositiveSmallIntegerField:
		return "PositiveSmallIntegerField"
	case orm.ORMTypePositiveIntegerField:
		return "PositiveIntegerField"
	case orm.ORMTypePositiveBigIntegerField:
		return "PositiveBigIntegerField"
	case orm.ORMTypeFloatField:
		return "FloatField"
	case orm.ORMTypeDecimalField:
		return "DecimalField"
	case orm.ORMTypeJSONField, orm.ORMTypeJsonbField:
		return "JSONField"
	case orm.ORMRelForeignKey:
		return "ForeignKey"
	case orm.ORMRelOneToOne:
		return "OneToOneField"
	case orm.ORMRelManyToMany:
		return "ManyToManyField"
	default:
		return "AutoField"
	}
}
