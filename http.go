package app

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"sync/atomic"
	"time"
)

type Request struct {
	id      uint64
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

var requestId uint64

func NewRequest(r *http.Request) Request {
	id := atomic.AddUint64(&requestId, 1)
	return Request{id: id, r: r, started: time.Now()}
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

func HTTPAdd[T Adder](seed func() T) func(Request) Response {
	return func(r Request) Response {
		decoder := json.NewDecoder(r.r.Body)
		a := seed()
		if err := decoder.Decode(&a); err != nil {
			r.Log("Could not decode data: %s", err)
			return r.jsonResponse(http.StatusBadRequest, formError("Petició mal formada"))
		}

		if errors := a.Validate(); len(errors) > 0 {
			return r.jsonResponse(http.StatusUnprocessableEntity, jsonErrors(errors))
		}

		if err := a.Add(); err != nil {
			return r.jsonResponse(http.StatusInternalServerError, formError(err.Error()))
		}

		return r.jsonResponse(http.StatusOK, map[string]any{"id": a.GetID()})
	}
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

func formError(msg string) map[string]ApiErrors {
	return jsonErrors(ApiErrors{"__form__": []string{msg}})
}

func jsonErrors(errors ApiErrors) map[string]ApiErrors {
	return map[string]ApiErrors{"errors": errors}
}
