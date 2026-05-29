package http

import (
	"io"
	"net/http"
)

type Response interface {
	Status() int
	InternalError() error
	Headers() map[string]string
	HasContent() bool
	WriteStatus(Request)
	io.Reader
}

type baseResponse struct {
	status      int
	internalErr error
	io.Reader
}

func newResponse(status int, err error, reader io.Reader) baseResponse {
	return baseResponse{status: status, internalErr: err, Reader: reader}
}

func (r baseResponse) Status() int                { return r.status }
func (r baseResponse) InternalError() error       { return r.internalErr }
func (r baseResponse) Headers() map[string]string { return nil }
func (r baseResponse) HasContent() bool           { return r.Reader != nil }

func (r baseResponse) WriteStatus(req Request) {
	req.w.WriteHeader(r.Status())
}

type JsonResponse struct {
	baseResponse
}

func (r JsonResponse) Headers() map[string]string {
	return map[string]string{
		"Content-Type": "application/json",
	}
}

func FixedJsonResponse(res any) func(Request) Response {
	return func(r Request) Response { return r.JsonResponse(res) }
}

type TemplateResponse struct {
	baseResponse
}

type RedirectResponse struct {
	baseResponse
	url string
}

func newRedirectResponse(url string) RedirectResponse {
	return RedirectResponse{newResponse(http.StatusFound, nil, nil), url}
}

func (r RedirectResponse) WriteStatus(req Request) {
	http.Redirect(req.w, req.r, r.url, r.status)
}
