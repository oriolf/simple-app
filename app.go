package app

// TODO Interesting approaches in diversos/temperatures
import (
	"bytes"
	"database/sql"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync/atomic"
	"time"
)

type Date struct {
	Year  uint
	Month time.Month
	Day   uint
}

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

var (
	db *sql.DB
)

type Option func() error

func InitSQL(migrationFiles embed.FS) Option {
	return func() (err error) {
		if db, err = initSQL(migrationFiles); err != nil {
			return fmt.Errorf("could not initialize sql: %w", err)
		}
		return nil
	}
}

func Init(options ...Option) (err error) {
	for _, opt := range options {
		if err := opt(); err != nil {
			return err
		}
	}
	return nil
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
		reader, writer := io.Pipe()
		go func() {
			if err := json.NewEncoder(writer).Encode(res); err != nil {
				r.Log("Could not encode response: %s", err)
			}
			writer.Close()
		}()
		return JsonResponse{status: http.StatusOK, Reader: reader}
	}
}

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

func internalServerErrorJsonResponse() Response {
	return JsonResponse{
		status: http.StatusInternalServerError,
		Reader: bytes.NewBuffer([]byte(`{"ok": false}`)),
	}
}
