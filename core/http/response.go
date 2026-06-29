// Package http mirrors Django's django.http module.
//
// Django:
//
//	from django.http import HttpResponse, JsonResponse, Http404
//	from django.http import HttpResponseRedirect, HttpResponsePermanentRedirect
//
// djanGO:
//
//	import "github.com/mqnifestkelvin/djanGO/core/http"
//
//	func MyView(w http.ResponseWriter, r *http.Request) {
//	    http.Ok(w, "Hello world")
//	    // or
//	    res := dhttp.NewResponse("Hello world")
//	    res.SetHeader("X-Custom", "value")
//	    res.Write(w)
//	}
package http

import (
	"encoding/json"
	"net/http"
)

// HttpResponse mirrors Django's HttpResponse.
// Wraps a status code, headers, content-type and body.
type HttpResponse struct {
	StatusCode  int
	headers     map[string]string
	ContentType string
	body        []byte
}

// NewResponse mirrors Django's HttpResponse(content, content_type, status).
func NewResponse(content string, args ...interface{}) *HttpResponse {
	r := &HttpResponse{
		StatusCode:  200,
		headers:     make(map[string]string),
		ContentType: "text/html; charset=utf-8",
		body:        []byte(content),
	}
	for _, arg := range args {
		switch v := arg.(type) {
		case int:
			r.StatusCode = v
		case string:
			r.ContentType = v
		}
	}
	return r
}

// SetHeader mirrors Django's response["Header-Name"] = value.
func (r *HttpResponse) SetHeader(key, value string) {
	r.headers[key] = value
}

// Write sends the response to the underlying http.ResponseWriter.
func (r *HttpResponse) Write(w http.ResponseWriter) {
	for k, v := range r.headers {
		w.Header().Set(k, v)
	}
	w.Header().Set("Content-Type", r.ContentType)
	w.WriteHeader(r.StatusCode)
	w.Write(r.body)
}

// HttpResponseRedirect mirrors Django's HttpResponseRedirect (302).
type HttpResponseRedirect struct {
	HttpResponse
	Location string
}

// NewRedirect mirrors Django's HttpResponseRedirect(redirect_to).
func NewRedirect(location string) *HttpResponseRedirect {
	r := &HttpResponseRedirect{
		HttpResponse: HttpResponse{
			StatusCode:  http.StatusFound, // 302
			headers:     make(map[string]string),
			ContentType: "text/html; charset=utf-8",
		},
		Location: location,
	}
	return r
}

func (r *HttpResponseRedirect) Write(w http.ResponseWriter) {
	for k, v := range r.headers {
		w.Header().Set(k, v)
	}
	http.Redirect(w, &http.Request{}, r.Location, r.StatusCode)
}

// HttpResponsePermanentRedirect mirrors Django's HttpResponsePermanentRedirect (301).
func NewPermanentRedirect(location string) *HttpResponseRedirect {
	r := NewRedirect(location)
	r.StatusCode = http.StatusMovedPermanently // 301
	return r
}

// JsonResponse mirrors Django's JsonResponse.
// Django: JsonResponse({"key": "value"})
type JsonResponse struct {
	HttpResponse
}

// NewJsonResponse mirrors Django's JsonResponse(data, safe=True, status=200).
func NewJsonResponse(data interface{}, status ...int) *JsonResponse {
	code := 200
	if len(status) > 0 {
		code = status[0]
	}
	body, _ := json.Marshal(data)
	return &JsonResponse{
		HttpResponse: HttpResponse{
			StatusCode:  code,
			headers:     make(map[string]string),
			ContentType: "application/json",
			body:        body,
		},
	}
}

// Http404 mirrors Django's Http404 exception.
// Raise it inside a view; the middleware catches it and returns a 404 response.
type Http404 struct {
	Message string
}

func (e *Http404) Error() string {
	if e.Message == "" {
		return "Not Found"
	}
	return e.Message
}

// --- Convenience helpers (mirrors django.http top-level usage) ---

// Ok writes a 200 text/html response — shorthand for HttpResponse(content).
func Ok(w http.ResponseWriter, content string) {
	NewResponse(content).Write(w)
}

// Json writes a 200 application/json response — shorthand for JsonResponse(data).
func Json(w http.ResponseWriter, data interface{}, status ...int) {
	NewJsonResponse(data, status...).Write(w)
}

// Redirect writes a 302 redirect — shorthand for HttpResponseRedirect(location).
func Redirect(w http.ResponseWriter, r *http.Request, location string) {
	http.Redirect(w, r, location, http.StatusFound)
}

// PermanentRedirect writes a 301 redirect — shorthand for HttpResponsePermanentRedirect.
func PermanentRedirect(w http.ResponseWriter, r *http.Request, location string) {
	http.Redirect(w, r, location, http.StatusMovedPermanently)
}

// NotFound writes a 404 response — mirrors Django's Http404 being caught by handler.
func NotFound(w http.ResponseWriter, message ...string) {
	msg := "Not Found"
	if len(message) > 0 {
		msg = message[0]
	}
	http.Error(w, msg, http.StatusNotFound)
}
