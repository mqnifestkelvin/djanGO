// Package forms mirrors Django's django.forms module.
//
// Django:
//
//	from django import forms
//
//	class ContactForm(forms.Form):
//	    name    = forms.CharField(max_length=100)
//	    email   = forms.EmailField()
//	    message = forms.CharField(widget=forms.Textarea)
//
// djanGO:
//
//	import "github.com/mqnifestkelvin/djanGO/core/forms"
//
//	type ContactForm struct {
//	    forms.Form
//	    Name    forms.CharField
//	    Email   forms.EmailField
//	    Message forms.CharField
//	}
package forms

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// ValidationError mirrors Django's ValidationError.
// Raised by field.Clean() when a value fails validation.
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string { return e.Message }

// Field is the interface every form field implements —
// mirrors Django's Field base class.
type Field interface {
	// Clean validates and coerces the raw string value from the HTTP request.
	// Mirrors Django's Field.clean(value).
	Clean(raw string) (interface{}, error)
	// IsRequired reports whether the field requires a non-empty value.
	IsRequired() bool
	// Label returns the human-readable label for the field.
	Label() string
}

// fieldBase holds common options shared by all field types —
// mirrors Django's Field.__init__ kwargs: required, label, initial, help_text.
type fieldBase struct {
	Required bool
	LabelText  string
	Initial   string
	HelpText  string
}

func (f *fieldBase) IsRequired() bool { return f.Required }
func (f *fieldBase) Label() string    { return f.LabelText }

func (f *fieldBase) checkRequired(raw string) error {
	if f.Required && strings.TrimSpace(raw) == "" {
		return &ValidationError{Message: "This field is required."}
	}
	return nil
}

// --- CharField ---

// CharField mirrors Django's forms.CharField.
//
// Django:
//
//	name = forms.CharField(max_length=100, required=True)
type CharField struct {
	fieldBase
	MaxLength int
	MinLength int
	Strip     bool
}

func NewCharField(opts ...func(*CharField)) CharField {
	f := CharField{
		fieldBase: fieldBase{Required: true},
		Strip:     true,
	}
	for _, o := range opts {
		o(&f)
	}
	return f
}

func (f *CharField) Clean(raw string) (interface{}, error) {
	if f.Strip {
		raw = strings.TrimSpace(raw)
	}
	if err := f.checkRequired(raw); err != nil {
		return nil, err
	}
	if f.MaxLength > 0 && len(raw) > f.MaxLength {
		return nil, &ValidationError{Message: fmt.Sprintf("Ensure this value has at most %d characters (it has %d).", f.MaxLength, len(raw))}
	}
	if f.MinLength > 0 && len(raw) < f.MinLength {
		return nil, &ValidationError{Message: fmt.Sprintf("Ensure this value has at least %d characters (it has %d).", f.MinLength, len(raw))}
	}
	return raw, nil
}

// --- EmailField ---

// EmailField mirrors Django's forms.EmailField.
//
// Django:
//
//	email = forms.EmailField()
var emailRe = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

type EmailField struct {
	fieldBase
}

func NewEmailField(opts ...func(*EmailField)) EmailField {
	f := EmailField{fieldBase: fieldBase{Required: true}}
	for _, o := range opts {
		o(&f)
	}
	return f
}

func (f *EmailField) Clean(raw string) (interface{}, error) {
	raw = strings.TrimSpace(raw)
	if err := f.checkRequired(raw); err != nil {
		return nil, err
	}
	if !emailRe.MatchString(raw) {
		return nil, &ValidationError{Message: "Enter a valid email address."}
	}
	return raw, nil
}

// --- IntegerField ---

// IntegerField mirrors Django's forms.IntegerField.
//
// Django:
//
//	age = forms.IntegerField(min_value=0, max_value=120)
type IntegerField struct {
	fieldBase
	MinValue *int
	MaxValue *int
}

func NewIntegerField(opts ...func(*IntegerField)) IntegerField {
	f := IntegerField{fieldBase: fieldBase{Required: true}}
	for _, o := range opts {
		o(&f)
	}
	return f
}

func (f *IntegerField) Clean(raw string) (interface{}, error) {
	raw = strings.TrimSpace(raw)
	if err := f.checkRequired(raw); err != nil {
		return nil, err
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return nil, &ValidationError{Message: "Enter a whole number."}
	}
	if f.MinValue != nil && n < *f.MinValue {
		return nil, &ValidationError{Message: fmt.Sprintf("Ensure this value is greater than or equal to %d.", *f.MinValue)}
	}
	if f.MaxValue != nil && n > *f.MaxValue {
		return nil, &ValidationError{Message: fmt.Sprintf("Ensure this value is less than or equal to %d.", *f.MaxValue)}
	}
	return n, nil
}

// --- BooleanField ---

// BooleanField mirrors Django's forms.BooleanField.
//
// Django:
//
//	agree = forms.BooleanField(required=True)
type BooleanField struct {
	fieldBase
}

func NewBooleanField(opts ...func(*BooleanField)) BooleanField {
	f := BooleanField{fieldBase: fieldBase{Required: false}}
	for _, o := range opts {
		o(&f)
	}
	return f
}

func (f *BooleanField) Clean(raw string) (interface{}, error) {
	raw = strings.ToLower(strings.TrimSpace(raw))
	val := raw != "" && raw != "false" && raw != "0"
	if f.Required && !val {
		return nil, &ValidationError{Message: "This field is required."}
	}
	return val, nil
}

// --- ChoiceField ---

// ChoiceField mirrors Django's forms.ChoiceField.
//
// Django:
//
//	color = forms.ChoiceField(choices=[("r","Red"),("g","Green")])
type ChoiceField struct {
	fieldBase
	Choices [][2]string // [value, label] pairs — mirrors Django's choices tuples
}

func NewChoiceField(choices [][2]string, opts ...func(*ChoiceField)) ChoiceField {
	f := ChoiceField{
		fieldBase: fieldBase{Required: true},
		Choices:   choices,
	}
	for _, o := range opts {
		o(&f)
	}
	return f
}

func (f *ChoiceField) Clean(raw string) (interface{}, error) {
	if err := f.checkRequired(raw); err != nil {
		return nil, err
	}
	for _, c := range f.Choices {
		if c[0] == raw {
			return raw, nil
		}
	}
	return nil, &ValidationError{Message: fmt.Sprintf("Select a valid choice. %s is not one of the available choices.", raw)}
}
