package http

import (
	"embed"
	"io"
	"log"
	"net/http"
)

var (
	defaultHeaders = map[string]string{
		"Access-Control-Allow-Origin":      "http://localhost:5173",
		"Access-Control-Allow-Credentials": "true",
		"Access-Control-Allow-Headers":     "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With",
		"Access-Control-Allow-Methods":     "POST, GET, OPTIONS, PUT, DELETE, PATCH, QUERY",
	}
)

func Serve() error {
	http.HandleFunc("OPTIONS /", func(w http.ResponseWriter, request *http.Request) {
		// TODO make it configurable
		for k, v := range defaultHeaders {
			w.Header().Add(k, v)
		}
	})

	port := ":8080"
	log.Printf("Listening on %s...\n", port)
	return http.ListenAndServe(port, nil)
}

func HandleIndex(files embed.FS) {
	http.HandleFunc("GET /{$}", StaticFile(files, "index.html"))
}

func StaticFile(staticFiles embed.FS, filename string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		http.ServeFileFS(w, r, staticFiles, "static/"+filename)
	}
}

func HandleStatic(url string, files embed.FS) {
	http.Handle(url, http.FileServerFS(files))
}

type httpGroup struct {
	options []option
}

func Group(options ...option) httpGroup {
	return httpGroup{options}
}

func (g httpGroup) Handle(url string, handler func(Request) Response) {
	Handle(url, handler, g.options...)
}

func Handle(url string, handler func(Request) Response, options ...option) {
	for _, option := range options {
		handler = option(handler)
	}

	http.HandleFunc(url, func(w http.ResponseWriter, request *http.Request) {
		r := NewRequest(request, w)
		r.Log("[%s] %s", request.Method, request.URL.Path)

		// TODO make it configurable
		for k, v := range defaultHeaders {
			w.Header().Add(k, v)
		}

		res := handler(r)
		if err := res.InternalError(); err != nil {
			r.Log("Got an error: %s", err)
		}

		for k, v := range res.Headers() {
			w.Header().Set(k, v)
		}

		res.WriteStatus(r)

		if res.HasContent() {
			if _, err := io.Copy(w, res); err != nil {
				r.Log("Could not send response: %s", err)
				return
			}
		}

		r.Log("[%d] %s", res.Status(), r.took())
	})
}
