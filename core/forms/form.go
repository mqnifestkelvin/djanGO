package forms

import (
	"net/http"
	"reflect"
	"strings"
)

// Form is the base struct every djanGO form embeds — mirrors Django's forms.Form.
//
// Django:
//
//	class ContactForm(forms.Form):
//	    name  = forms.CharField(max_length=100)
//	    email = forms.EmailField()
//
//	def view(request):
//	    form = ContactForm(request.POST)
//	    if form.is_valid():
//	        name = form.cleaned_data["name"]
//
// djanGO:
//
//	type ContactForm struct {
//	    forms.Form
//	    Name  forms.CharField
//	    Email forms.EmailField
//	}
//
//	func view(w http.ResponseWriter, r *http.Request) {
//	    form := forms.Bind[ContactForm](r)
//	    if form.IsValid() {
//	        name := form.CleanedData["name"]
//	    }
//	}
type Form struct {
	isBound     bool
	Errors      map[string]string      // mirrors form.errors — field name → error message
	CleanedData map[string]interface{} // mirrors form.cleaned_data
}

// IsValid mirrors Django's Form.is_valid().
// Returns true if the form is bound and has no errors.
//
// Django:
//
//	if form.is_valid():
//	    ...
func (f *Form) IsValid() bool {
	return f.isBound && len(f.Errors) == 0
}

// IsBound mirrors Django's Form.is_bound — True if data was submitted.
func (f *Form) IsBound() bool { return f.isBound }

// Bind populates a form struct from an HTTP request and runs validation —
// mirrors Django's Form(request.POST) constructor + is_valid() call pattern.
//
// Django:
//
//	form = ContactForm(request.POST or None)
//
// djanGO:
//
//	form := forms.Bind[ContactForm](r)
func Bind[T any](r *http.Request) *T {
	var form T

	// Parse form data from the request body
	_ = r.ParseForm()

	val := reflect.ValueOf(&form).Elem()
	typ := val.Type()

	errs := make(map[string]string)
	cleaned := make(map[string]interface{})
	isBound := r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch

	for i := 0; i < typ.NumField(); i++ {
		fieldVal := val.Field(i)
		fieldTyp := typ.Field(i)

		// Skip the embedded Form base
		if fieldTyp.Name == "Form" {
			continue
		}

		// Only process exported fields that implement Field
		if !fieldVal.CanInterface() {
			continue
		}

		iface := fieldVal.Addr().Interface()
		f, ok := iface.(Field)
		if !ok {
			continue
		}

		// Derive HTTP form key from field name — mirrors Django's field name lowercasing
		key := toSnakeCase(fieldTyp.Name)
		raw := r.FormValue(key)

		if isBound {
			cleanedVal, err := f.Clean(raw)
			if err != nil {
				errs[key] = err.Error()
			} else {
				cleaned[key] = cleanedVal
			}
		}
	}

	// Set the embedded Form fields via reflection
	formField := val.FieldByName("Form")
	if formField.IsValid() && formField.CanSet() {
		embedded := formField.Addr().Interface().(*Form)
		embedded.isBound = isBound
		embedded.Errors = errs
		embedded.CleanedData = cleaned
	}

	return &form
}

// toSnakeCase converts "FieldName" → "field_name" —
// mirrors Django's automatic form field name lowercasing.
func toSnakeCase(s string) string {
	var b strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			b.WriteByte('_')
		}
		b.WriteRune(r)
	}
	return strings.ToLower(b.String())
}
