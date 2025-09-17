package app

import (
	"bytes"
	"database/sql"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Request struct {
	id      uint
	r       *http.Request
	started time.Time
}

type Response interface {
	Status() int
	io.Reader
}

type JsonResponse struct {
	status int
	io.Reader
}

func (r JsonResponse) Status() int { return r.status }

type TemplateResponse struct {
	status int
	io.Reader
}

func (r TemplateResponse) Status() int { return r.status }

var requestId uint64

func NewRequest(r *http.Request) Request {
	id := atomic.AddUint64(&requestId, 1)
	return Request{id: uint(id), r: r, started: time.Now()}
}

func (r Request) took() time.Duration {
	return time.Since(r.started)
}

func (r Request) Log(msg string, args ...any) {
	args = append([]any{r.id}, args...)
	log.Printf("[%04d] "+msg+"\n", args...)
}

func ServeHTTP() error {
	defer db.Close()
	log.Println("Listening...")
	return http.ListenAndServe(":8080", nil)
}

func HandleHTTP(url string, handler func(Request) Response) {
	http.HandleFunc(url, func(w http.ResponseWriter, request *http.Request) {
		r := NewRequest(request)
		r.Log("[%s] %s", request.Method, request.URL.Path)

		res := handler(r)
		switch res.(type) {
		case JsonResponse:
			w.Header().Set("Content-Type", "application/json")
		}

		if st := res.Status(); st != http.StatusOK {
			w.WriteHeader(res.Status())
		}

		if _, err := io.Copy(w, res); err != nil {
			r.Log("Could not send response: %s", err)
			return
		}

		r.Log("[%d] %s", res.Status(), r.took())
	})
}

func FixedJsonResponse(res any) func(Request) Response {
	return func(r Request) Response {
		return r.jsonResponse(http.StatusOK, res)
	}
}

// TODO the template implementation is very limited: it only allows for one
// level of inheritance, everyone must inherit from layout.html
func HTTPTemplate(filename string) func(Request) Response {
	return func(r Request) Response {
		tmpl := getTemplate(filename)
		return r.templateResponse(tmpl, nil)
	}
}

func HTTPTemplateList[T Lister[T]](filename string, seed T) func(Request) Response {
	return func(r Request) Response {
		tmpl := getTemplate(filename)
		paginator := NewPaginator(r.r)
		items, total, err := seed.List(db, paginator)
		if err != nil {
			// TODO consider a nicer error.html page, with the error code, some message maybe
			// TODO consider also moving the error page to the simple-app code
			tmpl := getTemplate("error.html")
			return r.templateResponse(tmpl, nil)
		}
		paginator.SetTotal(total)
		return r.templateResponse(tmpl, map[string]any{"items": items, "total": total, "paginator": paginator})
	}
}

func HTTPAdd[T Adder](seed func() T) func(Request) Response {
	return func(r Request) Response {
		decoder := json.NewDecoder(r.r.Body)
		var params map[string]any
		if err := decoder.Decode(&params); err != nil {
			r.Log("Could not decode data: %s", err)
			return r.jsonResponse(http.StatusBadRequest, formError("Petició mal formada", nil))
		}

		a := seed()
		if errors := a.Validate(params); len(errors) > 0 {
			return r.jsonResponse(http.StatusUnprocessableEntity, jsonErrors(errors))
		}

		var id uint
		f := func(tx *sql.Tx) (err error) {
			id, err = a.Add(tx)
			return err
		}
		if err := transaction(db, f); err != nil {
			return r.jsonResponse(http.StatusInternalServerError, formError(err.Error(), a.ValidationTranslations()))
		}

		return r.jsonResponse(http.StatusOK, map[string]any{"id": id})
	}
}

func HTTPUpdate[T Updater](seed func() T) func(Request) Response {
	return func(r Request) Response {
		id, err := strconv.Atoi(r.r.PathValue("id"))
		if err != nil {
			return r.jsonResponse(http.StatusBadRequest, formError(err.Error(), nil))
		}

		decoder := json.NewDecoder(r.r.Body)
		var params map[string]any
		if err := decoder.Decode(&params); err != nil {
			r.Log("Could not decode data: %s", err)
			return r.jsonResponse(http.StatusBadRequest, formError("Petició mal formada", nil))
		}

		a := seed()
		a.SetID(uint(id))
		if errors := a.Validate(params); len(errors) > 0 {
			return r.jsonResponse(http.StatusUnprocessableEntity, jsonErrors(errors))
		}

		if err := transaction(db, a.Update); err != nil {
			return r.jsonResponse(http.StatusInternalServerError, formError(err.Error(), a.ValidationTranslations()))
		}

		return r.jsonResponse(http.StatusOK, map[string]any{})
	}
}

func HTTPPatch[T Patcher](seed T) func(Request) Response {
	return func(r Request) Response {
		id, err := strconv.Atoi(r.r.PathValue("id"))
		if err != nil {
			return r.jsonResponse(http.StatusBadRequest, formError(err.Error(), nil))
		}

		decoder := json.NewDecoder(r.r.Body)
		var params map[string]any
		if err := decoder.Decode(&params); err != nil {
			r.Log("Could not decode data: %s", err)
			return r.jsonResponse(http.StatusBadRequest, formError("Petició mal formada", nil))
		}

		if len(params) == 0 {
			return r.jsonResponse(http.StatusBadRequest, formError("Cal indicar el camp que es vol actualitzar", nil))
		}

		if len(params) > 1 {
			return r.jsonResponse(http.StatusBadRequest, formError("Només es pot actualitzar un camp per petició", nil))
		}

		var key string
		var value any
		for k, v := range params {
			key = k
			value = v
		}

		field, value, errors := seed.ValidatePatch(key, value)
		if len(errors) > 0 {
			return r.jsonResponse(http.StatusUnprocessableEntity, jsonErrors(errors))
		}

		f := func(tx *sql.Tx) error {
			return seed.Patch(tx, uint(id), field, value)
		}
		if err := transaction(db, f); err != nil {
			return r.jsonResponse(http.StatusInternalServerError, formError(err.Error(), seed.ValidationTranslations()))
		}

		return r.jsonResponse(http.StatusOK, map[string]any{})
	}
}

func HTTPDelete[T Deleter](seed T) func(Request) Response {
	return func(r Request) Response {
		id, err := strconv.Atoi(r.r.PathValue("id"))
		if err != nil {
			return r.jsonResponse(http.StatusBadRequest, formError(err.Error(), nil))
		}

		f := func(tx *sql.Tx) (err error) { return seed.Delete(tx, uint(id)) }
		if err := transaction(db, f); err != nil {
			return r.jsonResponse(http.StatusInternalServerError, formError(err.Error(), nil))
		}

		return r.jsonResponse(http.StatusOK, map[string]any{})
	}
}

func HTTPGet[T Getter[T]](seed T) func(Request) Response {
	return func(r Request) Response {
		id, err := strconv.Atoi(r.r.PathValue("id"))
		if err != nil {
			return r.jsonResponse(http.StatusBadRequest, formError(err.Error(), nil))
		}

		a, err := seed.Get(db, uint(id))
		if err != nil {
			return r.jsonResponse(http.StatusInternalServerError, formError(err.Error(), nil))
		}

		return r.jsonResponse(http.StatusOK, a)
	}
}

func HTTPList[T Lister[T]](seed T) func(Request) Response {
	return func(r Request) Response {
		items, total, err := seed.List(db, NewPaginator(r.r))
		if err != nil {
			return r.jsonResponse(http.StatusInternalServerError, formError(err.Error(), nil))
		}

		return r.jsonResponse(http.StatusOK, map[string]any{
			"total": total,
			"items": items,
		})
	}
}

func (r Request) templateResponse(tmpl *template.Template, data any) TemplateResponse {
	reader, writer := io.Pipe()
	go func() {
		if err := tmpl.ExecuteTemplate(writer, "layout.html", data); err != nil {
			r.Log("Could not execute template: %s", err)
		}
		writer.Close()
	}()
	return TemplateResponse{status: http.StatusOK, Reader: reader}
}

func (r Request) jsonResponse(status int, value any) JsonResponse {
	reader, writer := io.Pipe()
	go func() {
		if err := json.NewEncoder(writer).Encode(value); err != nil {
			r.Log("Could not encode response: %s", err)
		}
		writer.Close()
	}()
	return JsonResponse{status: status, Reader: reader}
}

func internalServerErrorJsonResponse() Response {
	return JsonResponse{
		status: http.StatusInternalServerError,
		Reader: bytes.NewBuffer([]byte(`{"ok": false}`)),
	}
}

func formError(msg string, translations map[string]string) map[string]ApiErrors {
	if translations != nil {
		for k, v := range translations {
			if strings.Contains(msg, k) {
				msg = v
				break
			}
		}
	}
	return jsonErrors(ApiErrors{"__form__": []string{msg}})
}

func jsonErrors(errors ApiErrors) map[string]ApiErrors {
	return map[string]ApiErrors{"errors": errors}
}

var (
	TEMPLATES      = make(map[string]*template.Template)
	TEMPLATES_LOCK sync.Mutex
)

func getTemplate(filename string) *template.Template {
	TEMPLATES_LOCK.Lock()
	defer TEMPLATES_LOCK.Unlock()
	return TEMPLATES[filename]
}

func initTemplates(templateFiles embed.FS) error {
	TEMPLATES_LOCK.Lock()
	defer TEMPLATES_LOCK.Unlock()

	files, err := templateFiles.ReadDir("templates")
	if err != nil {
		return fmt.Errorf("could not read dir: %w", err)
	}

	for _, f := range files {
		name := f.Name()
		tmpl, err := template.ParseFS(templateFiles, "templates/layout.html", "templates/"+name)
		if err != nil {
			return fmt.Errorf("could not parse template %s: %w", name, err)
		}
		TEMPLATES[name] = tmpl
	}

	return nil
}
