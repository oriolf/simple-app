package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync/atomic"
	"time"

	app "github.com/oriolf/simple-app"
	"github.com/oriolf/simple-app/users"
)

type Request struct {
	id      uint
	r       *http.Request
	w       http.ResponseWriter
	started time.Time

	User *users.User

	parameters map[string]any
}

var requestId uint64

func NewRequest(r *http.Request, w http.ResponseWriter) Request {
	id := atomic.AddUint64(&requestId, 1)
	return Request{id: uint(id), r: r, w: w, started: time.Now()}
}

func (r Request) took() time.Duration           { return time.Since(r.started) }
func (r Request) PathValue(field string) string { return r.r.PathValue(field) }
func (r Request) IP() string                    { return r.r.RemoteAddr }
func (r Request) Agent() string                 { return r.r.UserAgent() }

func (r Request) Log(msg string, args ...any) {
	args = append([]any{r.id}, args...)
	log.Printf("[%05d] "+msg+"\n", args...)
}

func (r Request) MustParameters() map[string]any {
	params, err := r.Parameters()
	if err != nil {
		panic(fmt.Sprintf("could not parse parameters: %s", err))
	}
	return params
}

func (r *Request) Parameters() (map[string]any, error) {
	if r.parameters != nil {
		return r.parameters, nil
	}

	params, err := r.parseParameters()
	if err != nil {
		return nil, err
	}

	r.parameters = params
	return params, nil
}

func (r Request) parseParameters() (map[string]any, error) {
	params := make(map[string]any)
	if r.r.Header.Get("Content-Type") == "application/json" {
		err := r.DecodeJsonBody(&params)
		if err != nil && !errors.Is(err, io.EOF) {
			return nil, fmt.Errorf("could not decode json body: %w", err)
		}

		if err == nil {
			return params, nil
		}
	}

	if err := r.r.ParseForm(); err != nil {
		return nil, fmt.Errorf("could not parse form: %w", err)
	}

	for k, lst := range r.r.Form {
		if len(lst) > 0 {
			params[k] = lst[0]
		}
	}

	return params, nil
}

func (r Request) TypedParameters(params any) error {
	return r.DecodeJsonBody(&params)
}

func (r Request) DecodeJsonBody(target any) error {
	decoder := json.NewDecoder(r.r.Body)
	return decoder.Decode(target)
}

func (r Request) JsonInternalError(msg string, err error) Response {
	return r.JsonGlobalError(http.StatusInternalServerError, msg, err)
}

func (r Request) JsonBadRequest(msg string, err error) Response {
	return r.JsonGlobalError(http.StatusBadRequest, msg, err)
}

func (r Request) JsonGlobalError(status int, msg string, err error) JsonResponse {
	return r.JsonError(status, app.NewGlobalApiError(msg), err)
}

func (r Request) JsonError(status int, userErrors app.ApiErrors, err error) JsonResponse {
	return r.jsonResponse(status, map[string]app.ApiErrors{"errors": userErrors}, err)
}

func (r Request) JsonResponse(value any) JsonResponse {
	return r.jsonResponse(http.StatusOK, value, nil)
}

func (r Request) jsonResponse(status int, value any, err error) JsonResponse {
	reader, writer := io.Pipe()
	go func() {
		if err := json.NewEncoder(writer).Encode(value); err != nil {
			r.Log("Could not encode response: %s", err)
		}
		writer.Close()
	}()
	return JsonResponse{newResponse(status, err, reader)}
}

func (r Request) TemplateError(err error) Response {
	return r.TemplateResponse("error.html", nil, err)
}

func (r Request) TemplateResponse(templateName string, data any, err error) TemplateResponse {
	tmpl := getTemplate(templateName)
	reader, writer := io.Pipe()
	go func() {
		if err := tmpl.ExecuteTemplate(writer, "layout.html", data); err != nil {
			r.Log("Could not execute template: %s", err)
		}
		writer.Close()
	}()
	return TemplateResponse{newResponse(http.StatusOK, err, reader)}
}
