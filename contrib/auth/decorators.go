package auth

import (
	"net/http"
	"net/url"

	"github.com/mqnifestkelvin/djanGO/core/middleware"
	"github.com/mqnifestkelvin/djanGO/core/urls"
)

// LoginRequired wraps a ViewFunc to require authentication —
// mirrors Django's @login_required decorator.
//
// Django:
//
//	from django.contrib.auth.decorators import login_required
//
//	@login_required
//	def my_view(request):
//	    ...
//
//	# Or with a custom login URL:
//	@login_required(login_url="/accounts/login/")
//	def my_view(request): ...
//
// Unauthenticated requests are redirected to LOGIN_URL with ?next=<current-path>.
func LoginRequired(view urls.ViewFunc) urls.ViewFunc {
	return loginRequiredWith(view, "/accounts/login/")
}

// LoginRequiredURL wraps a ViewFunc with a custom login URL —
// mirrors Django's @login_required(login_url=...).
func LoginRequiredURL(loginURL string) func(urls.ViewFunc) urls.ViewFunc {
	return func(view urls.ViewFunc) urls.ViewFunc {
		return loginRequiredWith(view, loginURL)
	}
}

func loginRequiredWith(view urls.ViewFunc, loginURL string) urls.ViewFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := middleware.UserFrom(r)
		if !user.IsAuthenticated {
			// Append ?next=<current-path> so the login view can redirect back.
			// Django: redirect_to = login_url + "?next=" + request.get_full_path()
			next := r.URL.RequestURI()
			redirectTo := loginURL + "?next=" + url.QueryEscape(next)
			http.Redirect(w, r, redirectTo, http.StatusFound)
			return
		}
		view(w, r)
	}
}

// StaffRequired wraps a ViewFunc to require is_staff=True —
// mirrors Django's @user_passes_test(lambda u: u.is_staff).
func StaffRequired(view urls.ViewFunc) urls.ViewFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := middleware.UserFrom(r)
		if !user.IsAuthenticated || !user.IsStaff {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		view(w, r)
	}
}
