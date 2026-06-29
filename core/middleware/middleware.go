// Package middleware mirrors Django's middleware system.
//
// Django middleware is a chain of callables that wrap the view handler.
// Each middleware can process the request before the view and the response after:
//
//	class MyMiddleware:
//	    def __init__(self, get_response):
//	        self.get_response = get_response
//
//	    def __call__(self, request):
//	        # process request
//	        response = self.get_response(request)
//	        # process response
//	        return response
//
// djanGO uses Go's standard http.Handler wrapping pattern — identical semantics:
//
//	func MyMiddleware(next http.Handler) http.Handler {
//	    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//	        // process request
//	        next.ServeHTTP(w, r)
//	        // process response (via wrapped ResponseWriter)
//	    })
// }
//
// Django applies middleware in order, innermost first (MIDDLEWARE list is reversed):
//
//	MIDDLEWARE = [
//	    "SecurityMiddleware",   ← outermost (runs first on request, last on response)
//	    "SessionMiddleware",
//	    "CommonMiddleware",
//	    ...
//	    "view",                 ← innermost
//	]
//
// djanGO mirrors this by wrapping in reverse order so the first entry in
// Middleware runs outermost.
package middleware

import "net/http"

// Func is a middleware function — takes the next handler and returns a new handler.
// Mirrors Django's middleware __init__(get_response) + __call__(request) pattern.
type Func func(next http.Handler) http.Handler

// Chain applies a list of middleware to a handler, outermost first —
// mirrors Django's load_middleware() which reverses MIDDLEWARE and wraps.
//
// Django applies [A, B, C] as A(B(C(view))) — A is outermost.
// We do the same: wrap in reverse so index 0 ends up outermost.
func Chain(handler http.Handler, middleware ...Func) http.Handler {
	for i := len(middleware) - 1; i >= 0; i-- {
		handler = middleware[i](handler)
	}
	return handler
}

// registry holds middleware registered by name — mirrors Django's dotted-path lookup.
var registry = make(map[string]Func)

// Register adds a named middleware to the registry.
// Django uses dotted import paths; djanGO uses short names like
// "djanGO.middleware.security.SecurityMiddleware".
func Register(name string, fn Func) {
	registry[name] = fn
}

// Lookup returns a registered middleware by its Django-style dotted name.
func Lookup(name string) (Func, bool) {
	fn, ok := registry[name]
	return fn, ok
}

// All returns all registered middleware names.
func All() map[string]Func {
	out := make(map[string]Func, len(registry))
	for k, v := range registry {
		out[k] = v
	}
	return out
}
