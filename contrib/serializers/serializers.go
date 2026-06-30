// Package serializers mirrors Django REST Framework's serializers.
//
// Django reference: rest_framework/serializers.py
//
// DRF serializers do two things:
//   1. Serialize: Go struct → map/JSON (for API responses)
//   2. Deserialize + validate: JSON/form data → Go struct (for API input)
//
// djanGO provides:
//   - Serializer     — base type, mirrors rest_framework.serializers.Serializer
//   - ModelSerializer[T] — auto-serializes any struct, mirrors ModelSerializer
//   - ToJSON()       — write a serialized response directly
//
// Usage:
//
//	// Define a serializer (mirrors DRF's Serializer with explicit fields)
//	type PostSerializer struct {
//	    serializers.Serializer
//	}
//
//	// Or use ModelSerializer for automatic field mapping:
//	s := serializers.NewModelSerializer(&post, serializers.SerializerMeta{
//	    Fields: "__all__",
//	})
//	data, _ := s.Data()
//	serializers.ToJSON(w, data)
//
// Django equivalent:
//
//	class PostSerializer(serializers.ModelSerializer):
//	    class Meta:
//	        model = Post
//	        fields = "__all__"
//
//	s = PostSerializer(post)
//	return Response(s.data)
package serializers

import (
	"encoding/json"
	"net/http"
	"reflect"
	"strings"
)

// ValidationError mirrors DRF's ValidationError.
// Returned when is_valid() fails.
//
// Django:
//
//	raise serializers.ValidationError("This field is required.")
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	if e.Field != "" {
		return e.Field + ": " + e.Message
	}
	return e.Message
}

// SerializerMeta mirrors DRF's Meta inner class on a ModelSerializer.
//
// Django:
//
//	class PostSerializer(serializers.ModelSerializer):
//	    class Meta:
//	        model = Post
//	        fields = "__all__"
//	        # or:
//	        fields = ["id", "title", "slug"]
//	        exclude = ["content"]
type SerializerMeta struct {
	Fields  string   // "__all__" or comma-separated field names
	Include []string // explicit include list (alternative to Fields string)
	Exclude []string // fields to exclude when Fields == "__all__"
}

// Serializer is the base type — embed this in custom serializers.
// Mirrors DRF's BaseSerializer / Serializer.
//
// Django:
//
//	class MySerializer(serializers.Serializer):
//	    title = serializers.CharField(max_length=200)
//	    email = serializers.EmailField()
//
//	s = MySerializer(data=request.data)
//	if s.is_valid():
//	    s.save()
type Serializer struct {
	Errors      map[string]string
	Fields      map[string]Field // declared fields — run during IsValid()
	initialData map[string]interface{}
	validated   map[string]interface{}
	createFn    func(data map[string]interface{}) (interface{}, error)
	updateFn    func(instance interface{}, data map[string]interface{}) (interface{}, error)
	instance    interface{}
}

// DeclareFields sets the field definitions used by IsValid() —
// call this in your serializer constructor to register validators.
//
// Django equivalent: declaring fields as class attributes on the serializer.
func (s *Serializer) DeclareFields(fields map[string]Field) {
	s.Fields = fields
}

// IsValid validates incoming data against declared fields —
// mirrors DRF's serializer.is_valid().
//
// Django:
//
//	s = MySerializer(data=request.data)
//	if s.is_valid():
//	    s.save()
//	else:
//	    return Response(s.errors, status=400)
func (s *Serializer) IsValid() bool {
	if s.Errors == nil {
		s.Errors = map[string]string{}
	}
	// If no field declarations, fall back to checking parse success only.
	if len(s.Fields) == 0 {
		return len(s.Errors) == 0
	}
	for name, field := range s.Fields {
		v, exists := s.initialData[name]
		if !exists || v == nil {
			if field.Required {
				s.Errors[name] = "this field is required"
			}
			continue
		}
		if field.validator != nil {
			if err := field.validator(v); err != nil {
				s.Errors[name] = err.Error()
			}
		}
	}
	if len(s.Errors) == 0 {
		s.validated = s.initialData
		return true
	}
	return false
}

// ValidatedData returns cleaned data after is_valid() — mirrors s.validated_data.
func (s *Serializer) ValidatedData() map[string]interface{} {
	return s.validated
}

// Save calls create() or update() depending on whether an instance was passed —
// mirrors DRF's serializer.save().
//
// Django:
//
//	s = PostSerializer(data=request.data)
//	if s.is_valid():
//	    post = s.save()           # calls create()
//
//	s = PostSerializer(post, data=request.data)
//	if s.is_valid():
//	    post = s.save()           # calls update()
func (s *Serializer) Save() (interface{}, error) {
	if s.instance != nil && s.updateFn != nil {
		return s.updateFn(s.instance, s.validated)
	}
	if s.createFn != nil {
		return s.createFn(s.validated)
	}
	return nil, nil
}

// SetCreate registers the create() hook — called by Save() when no instance.
// mirrors DRF's def create(self, validated_data).
func (s *Serializer) SetCreate(fn func(data map[string]interface{}) (interface{}, error)) {
	s.createFn = fn
}

// SetUpdate registers the update() hook — called by Save() when instance is set.
// mirrors DRF's def update(self, instance, validated_data).
func (s *Serializer) SetUpdate(fn func(instance interface{}, data map[string]interface{}) (interface{}, error)) {
	s.updateFn = fn
}

// SetInstance sets the instance for update operations.
func (s *Serializer) SetInstance(instance interface{}) {
	s.instance = instance
}

// ModelSerializer[T] auto-serializes any struct based on its exported fields.
// Mirrors DRF's ModelSerializer.
//
// Django:
//
//	class PostSerializer(serializers.ModelSerializer):
//	    class Meta:
//	        model = Post
//	        fields = "__all__"
type ModelSerializer[T any] struct {
	Serializer
	instance T
	meta     SerializerMeta
}

// NewModelSerializer creates a ModelSerializer for the given instance —
// mirrors PostSerializer(post).
//
// Django:
//
//	s = PostSerializer(post)
//	s.data  # → {"id": 1, "title": "Hello", ...}
func NewModelSerializer[T any](instance T, meta SerializerMeta) *ModelSerializer[T] {
	return &ModelSerializer[T]{instance: instance, meta: meta}
}

// Data serializes the instance to a map — mirrors serializer.data.
//
// Django:
//
//	s = PostSerializer(post)
//	s.data  # → OrderedDict([("id", 1), ("title", "Hello")])
func (s *ModelSerializer[T]) Data() (map[string]interface{}, error) {
	return structToMap(s.instance, s.meta)
}

// ManySerializer serializes a slice of structs — mirrors
// PostSerializer(posts, many=True).
//
// Django:
//
//	s = PostSerializer(posts, many=True)
//	s.data  # → [{"id": 1, ...}, {"id": 2, ...}]
type ManySerializer[T any] struct {
	instances []T
	meta      SerializerMeta
}

func NewManySerializer[T any](instances []T, meta SerializerMeta) *ManySerializer[T] {
	return &ManySerializer[T]{instances: instances, meta: meta}
}

func (s *ManySerializer[T]) Data() ([]map[string]interface{}, error) {
	result := make([]map[string]interface{}, 0, len(s.instances))
	for _, inst := range s.instances {
		m, err := structToMap(inst, s.meta)
		if err != nil {
			return nil, err
		}
		result = append(result, m)
	}
	return result, nil
}

// Bind deserializes the incoming JSON request body into a Serializer for validation —
// mirrors DRF's Serializer(data=request.data).
//
// Django:
//
//	s = PostSerializer(data=request.data)
//	if s.is_valid():
//	    post = s.save()
//
// After Bind(), call DeclareFields() then IsValid().
func Bind(r *http.Request) (*Serializer, error) {
	s := &Serializer{Errors: map[string]string{}}
	if err := json.NewDecoder(r.Body).Decode(&s.initialData); err != nil {
		s.Errors["non_field_errors"] = "Invalid JSON: " + err.Error()
		return s, nil
	}
	return s, nil
}

// BindWithInstance deserializes request body and attaches an existing instance
// for update — mirrors DRF's Serializer(instance, data=request.data).
//
// Django:
//
//	s = PostSerializer(post, data=request.data)
//	if s.is_valid():
//	    post = s.save()   # calls update()
func BindWithInstance(r *http.Request, instance interface{}) (*Serializer, error) {
	s, err := Bind(r)
	if err != nil {
		return s, err
	}
	s.instance = instance
	return s, nil
}

// ToJSON writes a serialized response as JSON —
// mirrors DRF's Response(serializer.data).
//
// Django:
//
//	return Response(serializer.data)
//	# → HTTP 200 with Content-Type: application/json
func ToJSON(w http.ResponseWriter, data interface{}, status ...int) {
	code := http.StatusOK
	if len(status) > 0 {
		code = status[0]
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(data)
}

// structToMap converts a struct to a map[string]interface{} using reflection,
// respecting the SerializerMeta include/exclude rules.
// Field names are snake_cased (mirrors DRF's default source mapping).
func structToMap(v interface{}, meta SerializerMeta) (map[string]interface{}, error) {
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return nil, nil
	}

	includeSet := buildIncludeSet(rv.Type(), meta)
	result := map[string]interface{}{}

	for i := 0; i < rv.Type().NumField(); i++ {
		f := rv.Type().Field(i)
		fv := rv.Field(i)

		// Skip unexported and embedded struct fields (like models.Model).
		if !f.IsExported() || (f.Anonymous && f.Type.Kind() == reflect.Struct) {
			continue
		}

		name := toSnakeCase(f.Name)
		// Respect json tag if present — mirrors DRF's source= kwarg.
		if tag := f.Tag.Get("json"); tag != "" {
			parts := strings.Split(tag, ",")
			if parts[0] != "" && parts[0] != "-" {
				name = parts[0]
			} else if parts[0] == "-" {
				continue
			}
		}

		if len(includeSet) > 0 {
			if _, ok := includeSet[name]; !ok {
				if _, ok2 := includeSet[f.Name]; !ok2 {
					continue
				}
			}
		}

		result[name] = fv.Interface()
	}
	return result, nil
}

// buildIncludeSet returns the set of field names to include.
// Empty map means include all (after excludes).
func buildIncludeSet(t reflect.Type, meta SerializerMeta) map[string]struct{} {
	excludeSet := map[string]struct{}{}
	for _, e := range meta.Exclude {
		excludeSet[e] = struct{}{}
		excludeSet[toSnakeCase(e)] = struct{}{}
	}

	if meta.Fields == "__all__" || (meta.Fields == "" && len(meta.Include) == 0) {
		// Include all exported non-embedded fields minus excludes.
		result := map[string]struct{}{}
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			if !f.IsExported() || (f.Anonymous && f.Type.Kind() == reflect.Struct) {
				continue
			}
			name := toSnakeCase(f.Name)
			if tag := f.Tag.Get("json"); tag != "" {
				parts := strings.Split(tag, ",")
				if parts[0] != "" && parts[0] != "-" {
					name = parts[0]
				}
			}
			if _, excluded := excludeSet[name]; !excluded {
				result[name] = struct{}{}
				result[f.Name] = struct{}{}
			}
		}
		return result
	}

	// Explicit field list.
	fields := meta.Include
	if meta.Fields != "" && meta.Fields != "__all__" {
		for _, f := range strings.Split(meta.Fields, ",") {
			fields = append(fields, strings.TrimSpace(f))
		}
	}
	result := map[string]struct{}{}
	for _, name := range fields {
		if _, excluded := excludeSet[name]; !excluded {
			result[name] = struct{}{}
			result[toSnakeCase(name)] = struct{}{}
		}
	}
	return result
}

// toSnakeCase converts Go field names to snake_case —
// mirrors DRF's automatic field name conversion.
//
// Handles acronyms correctly:
//   - "ID"          → "id"
//   - "URLField"    → "url_field"
//   - "HTTPSOn"     → "https_on"
//   - "PublishedAt" → "published_at"
//   - "CreatedAt"   → "created_at"
func toSnakeCase(s string) string {
	runes := []rune(s)
	n := len(runes)
	var b strings.Builder
	for i := 0; i < n; i++ {
		r := runes[i]
		if r >= 'A' && r <= 'Z' {
			// Insert underscore before this uppercase letter if:
			// - not at the start, AND
			// - previous char was lowercase (PublishedAt → published_at), OR
			// - next char is lowercase and previous is uppercase (URLField → url_field)
			if i > 0 {
				prev := runes[i-1]
				next := rune(0)
				if i+1 < n {
					next = runes[i+1]
				}
				if prev >= 'a' && prev <= 'z' {
					b.WriteByte('_')
				} else if prev >= 'A' && prev <= 'Z' && next >= 'a' && next <= 'z' {
					b.WriteByte('_')
				}
			}
			b.WriteRune(r + 32)
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}
