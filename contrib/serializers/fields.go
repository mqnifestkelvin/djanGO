package serializers

// Serializer fields — mirrors DRF's rest_framework/fields.py
//
// DRF:
//
//	class PostSerializer(serializers.Serializer):
//	    title = serializers.CharField(max_length=200, required=True)
//	    email = serializers.EmailField()
//	    age   = serializers.IntegerField(min_value=0, max_value=150)
//
// djanGO: build serializers with Field declarations, call IsValid() to run them.

import (
	"fmt"
	"net/mail"
	"regexp"
	"strconv"
	"strings"
)

// Field is a single validated serializer field — mirrors DRF's Field class.
type Field struct {
	Required  bool
	MaxLength int    // 0 = unlimited
	MinLength int
	MinValue  *int64 // nil = no minimum
	MaxValue  *int64 // nil = no maximum
	Pattern   string // regex pattern; empty = no check
	validator func(v interface{}) error
}

// CharField mirrors DRF's CharField.
//
// DRF:
//
//	title = serializers.CharField(max_length=200)
func CharField(opts ...func(*Field)) Field {
	f := &Field{Required: true}
	for _, o := range opts {
		o(f)
	}
	f.validator = func(v interface{}) error {
		s, ok := v.(string)
		if !ok {
			s = fmt.Sprintf("%v", v)
		}
		if f.Required && strings.TrimSpace(s) == "" {
			return fmt.Errorf("this field may not be blank")
		}
		if f.MaxLength > 0 && len(s) > f.MaxLength {
			return fmt.Errorf("ensure this field has no more than %d characters", f.MaxLength)
		}
		if f.MinLength > 0 && len(s) < f.MinLength {
			return fmt.Errorf("ensure this field has at least %d characters", f.MinLength)
		}
		if f.Pattern != "" {
			if matched, _ := regexp.MatchString(f.Pattern, s); !matched {
				return fmt.Errorf("this value does not match the required pattern")
			}
		}
		return nil
	}
	return *f
}

// EmailField mirrors DRF's EmailField.
//
// DRF:
//
//	email = serializers.EmailField()
func EmailField(opts ...func(*Field)) Field {
	f := &Field{Required: true}
	for _, o := range opts {
		o(f)
	}
	f.validator = func(v interface{}) error {
		s, _ := v.(string)
		if f.Required && s == "" {
			return fmt.Errorf("this field is required")
		}
		if s != "" {
			if _, err := mail.ParseAddress(s); err != nil {
				return fmt.Errorf("enter a valid email address")
			}
		}
		return nil
	}
	return *f
}

// IntegerField mirrors DRF's IntegerField.
//
// DRF:
//
//	age = serializers.IntegerField(min_value=0, max_value=150)
func IntegerField(opts ...func(*Field)) Field {
	f := &Field{Required: true}
	for _, o := range opts {
		o(f)
	}
	f.validator = func(v interface{}) error {
		var n int64
		switch val := v.(type) {
		case float64:
			n = int64(val)
		case int:
			n = int64(val)
		case int64:
			n = val
		case string:
			var err error
			n, err = strconv.ParseInt(val, 10, 64)
			if err != nil {
				return fmt.Errorf("a valid integer is required")
			}
		default:
			if f.Required {
				return fmt.Errorf("a valid integer is required")
			}
			return nil
		}
		if f.MinValue != nil && n < *f.MinValue {
			return fmt.Errorf("ensure this value is greater than or equal to %d", *f.MinValue)
		}
		if f.MaxValue != nil && n > *f.MaxValue {
			return fmt.Errorf("ensure this value is less than or equal to %d", *f.MaxValue)
		}
		return nil
	}
	return *f
}

// BooleanField mirrors DRF's BooleanField.
func BooleanField(opts ...func(*Field)) Field {
	f := &Field{Required: false}
	for _, o := range opts {
		o(f)
	}
	f.validator = func(v interface{}) error {
		switch v.(type) {
		case bool:
			return nil
		case string:
			s := strings.ToLower(v.(string))
			if s == "true" || s == "false" || s == "1" || s == "0" {
				return nil
			}
			return fmt.Errorf("must be a valid boolean")
		default:
			return nil
		}
	}
	return *f
}

// ---- Field option helpers (mirrors DRF keyword arguments) ----

// Required sets whether the field is required (default true).
func Required(required bool) func(*Field) {
	return func(f *Field) { f.Required = required }
}

// MaxLength sets max string length.
func MaxLength(n int) func(*Field) {
	return func(f *Field) { f.MaxLength = n }
}

// MinLength sets min string length.
func MinLength(n int) func(*Field) {
	return func(f *Field) { f.MinLength = n }
}

// MinValue sets the minimum numeric value.
func MinValue(n int64) func(*Field) {
	return func(f *Field) { f.MinValue = &n }
}

// MaxValue sets the maximum numeric value.
func MaxValue(n int64) func(*Field) {
	return func(f *Field) { f.MaxValue = &n }
}

// Pattern sets a regex validation pattern.
func Pattern(re string) func(*Field) {
	return func(f *Field) { f.Pattern = re }
}
