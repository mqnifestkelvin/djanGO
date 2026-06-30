package auth

// // Package auth — permissions subsystem.
//
// Mirrors Django's ModelBackend permission methods:
//   django/contrib/auth/backends.py  ModelBackend._get_permissions()
//   django/contrib/auth/mixins.py    PermissionRequiredMixin
//   django/contrib/auth/decorators.py permission_required
//
// How Django resolves has_perm("blog.add_post"):
//  1. Split "blog.add_post" → app_label="blog", codename="add_post"
//  2. Look up auth_permission WHERE codename=add_post AND content_type.app_label=blog
//  3. Check user_permissions OR groups→permissions for that row
//  4. Superusers always return True
//
// We use raw SQL so this package stays import-cycle-free.

import (
	"net/http"
	"strings"

	"github.com/mqnifestkelvin/djanGO/client/orm"
	"github.com/mqnifestkelvin/djanGO/core/middleware"
	"github.com/mqnifestkelvin/djanGO/core/urls"
)

// HasPerm checks whether user has the given permission string "app_label.codename" —
// mirrors Django's user.has_perm("blog.add_post").
//
// Django:
//
//	user.has_perm("blog.add_post")   # True/False
//	# Superusers always return True
//	# Anonymous users always return False
func (u *User) HasPerm(perm string) bool {
	if !u.IsActive {
		return false
	}
	if u.IsSuperuser {
		return true
	}

	appLabel, codename, ok := splitPerm(perm)
	if !ok {
		return false
	}

	o := orm.NewOrm()

	// Check direct user permissions (auth_user_user_permissions → auth_permission).
	var directCount int
	err := o.Raw(`
		SELECT COUNT(*) FROM auth_permission p
		JOIN django_content_type ct ON ct.id = p.content_type_id
		JOIN auth_user_user_permissions up ON up.permission_id = p.id
		WHERE up.user_id = ? AND p.codename = ? AND ct.app_label = ?
	`, u.Id, codename, appLabel).QueryRow(&directCount)
	if err == nil && directCount > 0 {
		return true
	}

	// Check group permissions (auth_user_groups → auth_group_permissions → auth_permission).
	var groupCount int
	err = o.Raw(`
		SELECT COUNT(*) FROM auth_permission p
		JOIN django_content_type ct ON ct.id = p.content_type_id
		JOIN auth_group_permissions gp ON gp.permission_id = p.id
		JOIN auth_user_groups ug ON ug.group_id = gp.group_id
		WHERE ug.user_id = ? AND p.codename = ? AND ct.app_label = ?
	`, u.Id, codename, appLabel).QueryRow(&groupCount)
	return err == nil && groupCount > 0
}

// HasPerms returns true if the user has all listed permissions —
// mirrors Django's user.has_perms(["blog.add_post", "blog.change_post"]).
func (u *User) HasPerms(perms []string) bool {
	for _, p := range perms {
		if !u.HasPerm(p) {
			return false
		}
	}
	return true
}

// GetAllPermissions returns the full set of "app_label.codename" strings for a user —
// mirrors Django's ModelBackend.get_all_permissions().
func (u *User) GetAllPermissions() ([]string, error) {
	if !u.IsActive {
		return nil, nil
	}
	o := orm.NewOrm()

	type row struct {
		AppLabel string
		Codename string
	}

	if u.IsSuperuser {
		var rows []orm.ParamsList
		_, err := o.Raw(`
			SELECT ct.app_label, p.codename FROM auth_permission p
			JOIN django_content_type ct ON ct.id = p.content_type_id
		`).ValuesList(&rows)
		if err != nil {
			return nil, err
		}
		return flattenRows(rows), nil
	}

	// Direct + group permissions union.
	var rows []orm.ParamsList
	_, err := o.Raw(`
		SELECT DISTINCT ct.app_label, p.codename FROM auth_permission p
		JOIN django_content_type ct ON ct.id = p.content_type_id
		WHERE p.id IN (
			SELECT permission_id FROM auth_user_user_permissions WHERE user_id = ?
			UNION
			SELECT gp.permission_id FROM auth_group_permissions gp
			JOIN auth_user_groups ug ON ug.group_id = gp.group_id
			WHERE ug.user_id = ?
		)
	`, u.Id, u.Id).ValuesList(&rows)
	if err != nil {
		return nil, err
	}
	return flattenRows(rows), nil
}

func flattenRows(rows []orm.ParamsList) []string {
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		if len(r) == 2 {
			out = append(out, r[0].(string)+"."+r[1].(string))
		}
	}
	return out
}

// splitPerm splits "app_label.codename" into its two parts.
func splitPerm(perm string) (appLabel, codename string, ok bool) {
	idx := strings.Index(perm, ".")
	if idx < 0 || idx == len(perm)-1 {
		return "", "", false
	}
	return perm[:idx], perm[idx+1:], true
}

// PermissionRequired wraps a view to require a specific permission —
// mirrors Django's @permission_required("blog.add_post").
//
// Django:
//
//	from django.contrib.auth.decorators import permission_required
//
//	@permission_required("blog.add_post")
//	def my_view(request): ...
//
//	# Unauthenticated → redirect to login; authenticated but lacking perm → 403
func PermissionRequired(perm string) func(urls.ViewFunc) urls.ViewFunc {
	return PermissionRequiredURL(perm, "/accounts/login/")
}

// PermissionRequiredURL is like PermissionRequired but with a custom login URL.
func PermissionRequiredURL(perm, loginURL string) func(urls.ViewFunc) urls.ViewFunc {
	return func(view urls.ViewFunc) urls.ViewFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			reqUser := middleware.UserFrom(r)
			if !reqUser.IsAuthenticated {
				http.Redirect(w, r, loginURL+"?next="+r.URL.RequestURI(), http.StatusFound)
				return
			}
			// Load the full User from DB so we can call HasPerm.
			o := orm.NewOrm()
			u := &User{}
			if err := o.QueryTable("auth_user").Filter("Id", reqUser.ID).One(u); err != nil {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}
			if !u.HasPerm(perm) {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}
			view(w, r)
		}
	}
}

// UserPassesTest wraps a view with a custom test function —
// mirrors Django's @user_passes_test(lambda u: u.is_staff).
//
// Django:
//
//	from django.contrib.auth.decorators import user_passes_test
//
//	@user_passes_test(lambda u: u.is_staff)
//	def my_view(request): ...
func UserPassesTest(testFunc func(*middleware.RequestUser) bool) func(urls.ViewFunc) urls.ViewFunc {
	return func(view urls.ViewFunc) urls.ViewFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			user := middleware.UserFrom(r)
			if !testFunc(user) {
				if !user.IsAuthenticated {
					http.Redirect(w, r, "/accounts/login/?next="+r.URL.RequestURI(), http.StatusFound)
					return
				}
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}
			view(w, r)
		}
	}
}
