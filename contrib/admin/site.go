package admin

// AdminSite mirrors Django's django.contrib.admin.AdminSite.
//
// Django:
//
//	# django/contrib/admin/sites.py
//	class AdminSite:
//	    def register(self, model_or_iterable, admin_class=None, **options): ...
//	    def get_urls(self): ...
//
//	# default instance used by apps:
//	site = AdminSite()
//
// djanGO:
//
//	admin.Site.Register("blog", "post", admin.ModelAdmin{...})
//	// then in urls.go:
//	urls.Path("/admin/", admin.Site.URLs())

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"sort"
	"strings"

	"github.com/mqnifestkelvin/djanGO/client/orm"
	"github.com/mqnifestkelvin/djanGO/contrib/contenttypes"
	"github.com/mqnifestkelvin/djanGO/core/middleware"
	"github.com/mqnifestkelvin/djanGO/core/urls"
)

// ModelAdmin holds per-model admin configuration —
// mirrors Django's ModelAdmin class.
//
// Django:
//
//	class PostAdmin(admin.ModelAdmin):
//	    list_display = ["title", "published", "created_at"]
//	    search_fields = ["title", "content"]
type ModelAdmin struct {
	ListDisplay  []string // columns shown in list view (default: ["id"])
	SearchFields []string // fields searched by ?q= parameter
}

type registration struct {
	appLabel  string
	modelName string
	admin     ModelAdmin
	newFn     func() interface{} // factory to create empty model instance
}

// AdminSite is the central registry — mirrors Django's AdminSite.
type AdminSite struct {
	registrations []registration
	prefix        string
}

// Site is the default admin site — mirrors Django's `admin.site`.
//
// Django:
//
//	from django.contrib import admin
//	admin.site.register(Post)
var Site = &AdminSite{prefix: "/admin/"}

// Register adds a model to the admin site —
// mirrors admin.site.register(Post) / admin.site.register(Post, PostAdmin).
//
// appLabel: e.g. "blog"
// modelName: e.g. "post" (lowercase)
// newFn: factory returning a new empty model pointer, e.g. func() interface{} { return &Post{} }
// ma: optional ModelAdmin config
//
// Django:
//
//	admin.site.register(Post)
//	admin.site.register(Post, PostAdmin)
func (s *AdminSite) Register(appLabel, modelName string, newFn func() interface{}, ma ModelAdmin) {
	if len(ma.ListDisplay) == 0 {
		ma.ListDisplay = []string{"id"}
	}
	s.registrations = append(s.registrations, registration{
		appLabel:  appLabel,
		modelName: modelName,
		admin:     ma,
		newFn:     newFn,
	})
}

// URLs returns the http.Handler for the entire admin site —
// mount this at /admin/ in your urls.go.
//
// Django:
//
//	# urls.py
//	path("admin/", admin.site.urls)
func (s *AdminSite) URLs() urls.ViewFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Strip the /admin/ prefix so sub-routes are relative.
		path := strings.TrimPrefix(r.URL.Path, s.prefix)
		path = "/" + strings.TrimPrefix(path, "/")

		if !requireStaff(w, r) {
			return
		}

		// Route: /admin/  → index
		if path == "/" || path == "" {
			s.indexView(w, r)
			return
		}

		// Route: /admin/<app>/<model>/           → list
		// Route: /admin/<app>/<model>/add/       → add form
		// Route: /admin/<app>/<model>/<id>/      → change form
		// Route: /admin/<app>/<model>/<id>/delete/ → delete confirm
		parts := strings.Split(strings.Trim(path, "/"), "/")
		if len(parts) < 2 {
			http.NotFound(w, r)
			return
		}
		appLabel := parts[0]
		modelName := parts[1]
		reg, ok := s.findReg(appLabel, modelName)
		if !ok {
			http.NotFound(w, r)
			return
		}

		switch {
		case len(parts) == 2 || (len(parts) == 3 && parts[2] == ""):
			s.listView(w, r, reg)
		case len(parts) == 3 && parts[2] == "add":
			s.addView(w, r, reg)
		case len(parts) == 4 && parts[3] == "delete":
			s.deleteView(w, r, reg, parts[2])
		case len(parts) == 3:
			s.changeView(w, r, reg, parts[2])
		default:
			http.NotFound(w, r)
		}
	}
}

func (s *AdminSite) findReg(app, model string) (registration, bool) {
	for _, r := range s.registrations {
		if r.appLabel == app && r.modelName == model {
			return r, true
		}
	}
	return registration{}, false
}

// requireStaff checks is_staff — mirrors Django's AdminSite.has_permission().
func requireStaff(w http.ResponseWriter, r *http.Request) bool {
	u := middleware.UserFrom(r)
	if !u.IsAuthenticated {
		http.Redirect(w, r, "/admin/login/?next="+r.URL.RequestURI(), http.StatusFound)
		return false
	}
	if !u.IsStaff {
		http.Error(w, "Forbidden: staff access required", http.StatusForbidden)
		return false
	}
	return true
}

// indexView renders the admin index — lists all registered models.
func (s *AdminSite) indexView(w http.ResponseWriter, r *http.Request) {
	type modelEntry struct {
		AppLabel  string
		ModelName string
		AddURL    string
		ListURL   string
	}
	var entries []modelEntry
	for _, reg := range s.registrations {
		base := fmt.Sprintf("/admin/%s/%s/", reg.appLabel, reg.modelName)
		entries = append(entries, modelEntry{
			AppLabel:  reg.appLabel,
			ModelName: reg.modelName,
			AddURL:    base + "add/",
			ListURL:   base,
		})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].AppLabel != entries[j].AppLabel {
			return entries[i].AppLabel < entries[j].AppLabel
		}
		return entries[i].ModelName < entries[j].ModelName
	})
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"site":   "djanGO admin",
		"models": entries,
	})
}

// listView lists all objects for a model — mirrors Django's ChangeList view.
func (s *AdminSite) listView(w http.ResponseWriter, r *http.Request, reg registration) {
	o := orm.NewOrm()
	table := reg.modelName

	// Try to get table name from a real instance.
	inst := reg.newFn()
	if tn, ok := inst.(interface{ TableName() string }); ok {
		table = tn.TableName()
	}

	q := r.URL.Query().Get("q")
	query := fmt.Sprintf("SELECT * FROM `%s`", table)
	args := []interface{}{}

	if q != "" && len(reg.admin.SearchFields) > 0 {
		var clauses []string
		for _, f := range reg.admin.SearchFields {
			clauses = append(clauses, fmt.Sprintf("`%s` LIKE ?", f))
			args = append(args, "%"+q+"%")
		}
		query += " WHERE " + strings.Join(clauses, " OR ")
	}

	var rows []orm.ParamsList
	_, err := o.Raw(query, args...).ValuesList(&rows)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"model": reg.appLabel + "." + reg.modelName,
		"rows":  rows,
	})
}

// addView handles POST to create a new object — mirrors Django's add_view.
func (s *AdminSite) addView(w http.ResponseWriter, r *http.Request, reg registration) {
	if r.Method == http.MethodGet {
		// Return the list of fields for the form.
		inst := reg.newFn()
		fields := structFieldNames(inst)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"model":  reg.appLabel + "." + reg.modelName,
			"fields": fields,
		})
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	inst := reg.newFn()
	if err := json.NewDecoder(r.Body).Decode(inst); err != nil {
		http.Error(w, "Bad Request: "+err.Error(), http.StatusBadRequest)
		return
	}

	o := orm.NewOrm()
	if _, err := o.Insert(inst); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Log the action.
	u := middleware.UserFrom(r)
	ct, _ := contenttypes.GetForModel(reg.appLabel, reg.modelName)
	ctID := 0
	if ct != nil {
		ctID = ct.Id
	}
	_ = LogAction(u.ID, ctID, fmt.Sprintf("%v", pkOf(inst)), fmt.Sprintf("%v", inst), ActionAddition, "")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(inst)
}

// changeView handles GET (show form) and POST (update) for a single object.
func (s *AdminSite) changeView(w http.ResponseWriter, r *http.Request, reg registration, id string) {
	o := orm.NewOrm()
	inst := reg.newFn()

	table := reg.modelName
	if tn, ok := inst.(interface{ TableName() string }); ok {
		table = tn.TableName()
	}

	var rows []orm.ParamsList
	_, err := o.Raw(fmt.Sprintf("SELECT * FROM `%s` WHERE id=?", table), id).ValuesList(&rows)
	if err != nil || len(rows) == 0 {
		http.NotFound(w, r)
		return
	}

	if r.Method == http.MethodGet {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(rows[0])
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := json.NewDecoder(r.Body).Decode(inst); err != nil {
		http.Error(w, "Bad Request: "+err.Error(), http.StatusBadRequest)
		return
	}

	if _, err := o.Update(inst); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	u := middleware.UserFrom(r)
	ct, _ := contenttypes.GetForModel(reg.appLabel, reg.modelName)
	ctID := 0
	if ct != nil {
		ctID = ct.Id
	}
	_ = LogAction(u.ID, ctID, id, fmt.Sprintf("%v", inst), ActionChange, "")

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(inst)
}

// deleteView handles GET (confirm page) and POST (confirm delete).
func (s *AdminSite) deleteView(w http.ResponseWriter, r *http.Request, reg registration, id string) {
	o := orm.NewOrm()
	inst := reg.newFn()

	table := reg.modelName
	if tn, ok := inst.(interface{ TableName() string }); ok {
		table = tn.TableName()
	}

	if r.Method == http.MethodGet {
		var rows []orm.ParamsList
		_, _ = o.Raw(fmt.Sprintf("SELECT * FROM `%s` WHERE id=?", table), id).ValuesList(&rows)
		w.Header().Set("Content-Type", "application/json")
		if len(rows) == 0 {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"confirm": fmt.Sprintf("DELETE %s.%s id=%s?", reg.appLabel, reg.modelName, id),
			"object":  rows[0],
		})
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	if _, err := o.Raw(fmt.Sprintf("DELETE FROM `%s` WHERE id=?", table), id).Exec(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	u := middleware.UserFrom(r)
	ct, _ := contenttypes.GetForModel(reg.appLabel, reg.modelName)
	ctID := 0
	if ct != nil {
		ctID = ct.Id
	}
	_ = LogAction(u.ID, ctID, id, reg.modelName+" #"+id, ActionDeletion, "")

	w.WriteHeader(http.StatusNoContent)
}

// structFieldNames returns exported field names of a struct pointer.
func structFieldNames(v interface{}) []string {
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	var names []string
	for i := 0; i < rv.Type().NumField(); i++ {
		f := rv.Type().Field(i)
		if f.IsExported() && !f.Anonymous {
			names = append(names, f.Name)
		}
	}
	return names
}

// pkOf attempts to read the Id field from a struct pointer.
func pkOf(v interface{}) interface{} {
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	f := rv.FieldByName("Id")
	if f.IsValid() {
		return f.Interface()
	}
	return 0
}
