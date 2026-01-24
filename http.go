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
	"sync"
	"sync/atomic"
	"time"
)

type Request struct {
	id      uint
	r       *http.Request
	w       http.ResponseWriter
	started time.Time
	DB      *sql.DB
	User    *User
}

var requestId uint64

func NewRequest(r *http.Request, w http.ResponseWriter) Request {
	id := atomic.AddUint64(&requestId, 1)
	return Request{id: uint(id), r: r, w: w, started: time.Now(), DB: db}
}

func (r Request) Log(msg string, args ...any) {
	args = append([]any{r.id}, args...)
	log.Printf("[%04d] "+msg+"\n", args...)
}

func (r Request) took() time.Duration           { return time.Since(r.started) }
func (r Request) PathValue(field string) string { return r.r.PathValue(field) }
func (r Request) IP() string                    { return r.r.RemoteAddr }
func (r Request) Agent() string                 { return r.r.UserAgent() }

type Response interface {
	Status() int
	Error() error
	io.Reader
}

type JsonResponse struct {
	status int
	err    error
	io.Reader
}

func (r JsonResponse) Status() int  { return r.status }
func (r JsonResponse) Error() error { return r.err }

type TemplateResponse struct {
	err    error
	status int
	io.Reader
}

func (r TemplateResponse) Status() int  { return r.status }
func (r TemplateResponse) Error() error { return r.err }

type RedirectResponse struct {
	status int
	io.Reader
}

func (r RedirectResponse) Status() int  { return r.status }
func (r RedirectResponse) Error() error { return nil }

func ServeHTTP() error {
	http.HandleFunc("OPTIONS /", func(w http.ResponseWriter, request *http.Request) {
		// TODO make it configurable
		w.Header().Add("Access-Control-Allow-Origin", "http://localhost:5173")
		w.Header().Add("Access-Control-Allow-Credentials", "true")
		w.Header().Add("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE, PATCH, QUERY")
	})

	defer db.Close()
	port := ":8080"
	log.Printf("Listening on %s...\n", port)
	return http.ListenAndServe(port, nil)
}

func HTTPStatic(staticFiles embed.FS, filename string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		http.ServeFileFS(w, r, staticFiles, "static/"+filename)
	}
}

func (g httpGroup) HandleHTTP(url string, handler func(Request) Response) {
	HandleHTTP(url, handler, g.options...)
}

func HandleHTTP(url string, handler func(Request) Response, options ...httpOption) {
	http.HandleFunc(url, func(w http.ResponseWriter, request *http.Request) {
		r := NewRequest(request, w)
		r.Log("[%s] %s", request.Method, request.URL.Path)

		// TODO make it configurable
		w.Header().Add("Access-Control-Allow-Origin", "http://localhost:5173")
		w.Header().Add("Access-Control-Allow-Credentials", "true")
		w.Header().Add("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		w.Header().Add("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE, PATCH, QUERY")

		var err error
		var res Response
		authenticator := getHttpAuthenticator(options...)
		r.User, err = authenticator(r)
		if err != nil {
			r.Log("Got an authentication error: %s", err)
			res = r.jsonResponse(http.StatusUnauthorized, formError("", r.User), err)
		} else {
			res = handler(r)
			if err := res.Error(); err != nil {
				r.Log("Got an error: %s", err)
			}
		}

		switch res.(type) {
		case JsonResponse:
			w.Header().Set("Content-Type", "application/json")
		}

		if st := res.Status(); st != http.StatusOK && st != http.StatusFound {
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
		return r.jsonResponse(http.StatusOK, res, nil)
	}
}

func HTTPTemplate(filename string) func(Request) Response {
	return func(r Request) Response {
		return r.TemplateResponse(filename, nil, nil)
	}
}

func HTTPTemplateList[C any, T Lister[T, C]](filename string, seed T) func(Request) Response {
	return func(r Request) Response {
		paginator := NewPaginator(r.r)
		items, total, err := seed.List(db, paginator, seed.FilterCriteria(r.r))
		if err != nil {
			return r.TemplateError(err)
		}
		paginator.SetTotal(total)
		return r.TemplateResponse(filename, map[string]any{"items": items, "total": total, "paginator": paginator}, nil)
	}
}

func (r Request) TemplateError(err error) Response {
	return r.TemplateResponse("error.html", nil, err)
}

func Login(r Request) Response {
	var u User
	params, err := getHttpDecoder()(r)
	if err != nil {
		r.Log("Could not decode data: %s", err)
		return r.jsonResponse(http.StatusBadRequest, formError("Petició mal formada", u), err)
	}

	if errors := u.Validate(params); len(errors) > 0 {
		return r.jsonResponse(http.StatusUnprocessableEntity, jsonErrors(errors), nil)
	}

	u, err = DBGetBy(r.DB, u, "email", u.Email, struct{}{})
	if err != nil {
		return r.jsonResponse(http.StatusBadRequest, formError("Correu o contrasenya incorrectes", u), nil)
	}

	providedPassword := hashPassword(u.Salt, params["password"].(string))
	if providedPassword != u.Password {
		return r.jsonResponse(http.StatusBadRequest, formError("Correu o contrasenya incorrectes", u), nil)
	}

	// TODO make cookies more secure: https://www.calhoun.io/securing-cookies-in-go
	s := NewSession(u.ID, r)
	f := func(tx *sql.Tx) (err error) {
		_, err = DBAdd(tx, s)
		return err
	}
	if err := transaction(db, f); err != nil {
		return r.jsonResponse(http.StatusInternalServerError, formError("", u), err)
	}
	c := http.Cookie{
		Name:  "_session",
		Value: s.ID,
	}
	http.SetCookie(r.w, &c)

	return getHttpReturner()(r, http.StatusOK, map[string]any{"ok": true})
}

func NewSession(userID uint, r Request) Session {
	return Session{
		ID:      generateRandomID(),
		UserID:  userID,
		Time:    Now(),
		Expires: Now().Add(30 * 24 * time.Hour),
		IP:      r.IP(),
		Agent:   r.Agent(),
	}
}

func Me(r Request) Response {
	s := Session{UserID: r.User.ID}
	sessions, _, err := DBList(r.DB, s, nil, struct{}{})
	if err != nil {
		return r.jsonResponse(http.StatusInternalServerError, formError("", s), err)
	}

	r.User.Sessions = sessions
	return getHttpReturner()(r, http.StatusOK, r.User)
}

func DeleteSession(r Request) Response {
	id := r.r.PathValue("id")
	var userID uint
	err := r.DB.QueryRow("SELECT user_id FROM sessions WHERE id=?;", id).Scan(&userID)
	if err != nil {
		return r.jsonResponse(http.StatusNotFound, formError(err.Error(), r.User), err)
	}

	if userID != r.User.ID {
		return r.jsonResponse(http.StatusNotFound, formError("La sessió no existeix", r.User), nil)
	}

	if _, err := db.Exec("DELETE FROM sessions WHERE id=?;", id); err != nil {
		return r.jsonResponse(http.StatusInternalServerError, formError(err.Error(), r.User), err)
	}

	return r.jsonResponse(http.StatusOK, map[string]any{"ok": true}, nil)
}

func HTTPAdd[T Adder](seed func() T, options ...httpOption) func(Request) Response {
	return func(r Request) Response {
		decoder := getHttpDecoder(options...)
		params, err := decoder(r)
		a := seed()
		if err != nil {
			r.Log("Could not decode data: %s", err)
			return r.jsonResponse(http.StatusBadRequest, formError("Petició mal formada", a), err)
		}

		if errors := a.Validate(params); len(errors) > 0 {
			return r.jsonResponse(http.StatusUnprocessableEntity, jsonErrors(errors), nil)
		}

		var id uint
		f := func(tx *sql.Tx) (err error) {
			id, err = a.Add(tx)
			return err
		}
		if err := transaction(db, f); err != nil {
			return r.jsonResponse(http.StatusInternalServerError, formError(err.Error(), a), err)
		}

		returner := getHttpReturner(options...)
		return returner(r, http.StatusOK, map[string]any{"id": id})
	}
}

func HTTPQueryAdd[T Adder](seed func() T, options ...httpOption) func(Request) Response {
	return func(r Request) Response {
		decoder := getHttpDecoder(options...)
		params, err := decoder(r)
		a := seed()
		if err != nil {
			r.Log("Could not decode data: %s", err)
			return r.jsonResponse(http.StatusBadRequest, formError("Petició mal formada", a), err)
		}

		if errors := a.Validate(params); len(errors) > 0 {
			return r.jsonResponse(http.StatusUnprocessableEntity, jsonErrors(errors), nil)
		}

		returner := getHttpReturner(options...)
		return returner(r, http.StatusOK, map[string]any{"ok": true})
	}
}

func HTTPUpdate[T Updater](seed func() T) func(Request) Response {
	return func(r Request) Response {
		id, err := strconv.Atoi(r.r.PathValue("id"))
		a := seed()
		if err != nil {
			return r.jsonResponse(http.StatusBadRequest, formError(err.Error(), a), err)
		}

		decoder := json.NewDecoder(r.r.Body)
		var params map[string]any
		if err := decoder.Decode(&params); err != nil {
			r.Log("Could not decode data: %s", err)
			return r.jsonResponse(http.StatusBadRequest, formError("Petició mal formada", a), err)
		}

		a.SetID(uint(id))
		if errors := a.Validate(params); len(errors) > 0 {
			return r.jsonResponse(http.StatusUnprocessableEntity, jsonErrors(errors), nil)
		}

		if err := transaction(db, a.Update); err != nil {
			return r.jsonResponse(http.StatusInternalServerError, formError(err.Error(), a), err)
		}

		return r.jsonResponse(http.StatusOK, map[string]any{}, nil)
	}
}

func HTTPPatch[T Patcher](seed T) func(Request) Response {
	return func(r Request) Response {
		id, err := strconv.Atoi(r.r.PathValue("id"))
		if err != nil {
			return r.jsonResponse(http.StatusBadRequest, formError(err.Error(), seed), err)
		}

		decoder := json.NewDecoder(r.r.Body)
		var params map[string]any
		if err := decoder.Decode(&params); err != nil {
			r.Log("Could not decode data: %s", err)
			return r.jsonResponse(http.StatusBadRequest, formError("Petició mal formada", seed), err)
		}

		if len(params) == 0 {
			return r.jsonResponse(http.StatusBadRequest, formError("Cal indicar el camp que es vol actualitzar", seed), nil)
		}

		if len(params) > 1 {
			return r.jsonResponse(http.StatusBadRequest, formError("Només es pot actualitzar un camp per petició", seed), nil)
		}

		var key string
		var value any
		for k, v := range params {
			key = k
			value = v
		}

		field, value, errors := seed.ValidatePatch(key, value)
		if len(errors) > 0 {
			return r.jsonResponse(http.StatusUnprocessableEntity, jsonErrors(errors), nil)
		}

		f := func(tx *sql.Tx) error {
			return seed.Patch(tx, uint(id), field, value)
		}
		if err := transaction(db, f); err != nil {
			return r.jsonResponse(http.StatusInternalServerError, formError(err.Error(), seed), err)
		}

		return r.jsonResponse(http.StatusOK, map[string]any{}, nil)
	}
}

func HTTPDelete[T Deleter](seed T) func(Request) Response {
	return func(r Request) Response {
		id, err := strconv.Atoi(r.r.PathValue("id"))
		if err != nil {
			return r.jsonResponse(http.StatusBadRequest, formError(err.Error(), seed), err)
		}

		f := func(tx *sql.Tx) (err error) { return seed.Delete(tx, uint(id)) }
		if err := transaction(db, f); err != nil {
			return r.jsonResponse(http.StatusInternalServerError, formError(err.Error(), seed), err)
		}

		return r.jsonResponse(http.StatusOK, map[string]any{}, nil)
	}
}

func HTTPGet[T Getter[T]](seed T) func(Request) Response {
	return func(r Request) Response {
		id, err := strconv.Atoi(r.r.PathValue("id"))
		if err != nil {
			return r.jsonResponse(http.StatusBadRequest, formError(err.Error(), seed), err)
		}

		a, err := seed.Get(db, uint(id))
		if err != nil {
			return r.jsonResponse(http.StatusInternalServerError, formError(err.Error(), seed), err)
		}

		return r.jsonResponse(http.StatusOK, a, nil)
	}
}

func HTTPList[C any, T Lister[T, C]](seed T) func(Request) Response {
	return func(r Request) Response {
		items, total, err := seed.List(db, NewPaginator(r.r), seed.FilterCriteria(r.r))
		if err != nil {
			return r.jsonResponse(http.StatusInternalServerError, formError(err.Error(), seed), err)
		}

		return r.jsonResponse(http.StatusOK, map[string]any{
			"total": total,
			"items": items,
		}, nil)
	}
}

func getHttpDecoder(options ...httpOption) func(r Request) (map[string]any, error) {
	for _, option := range options {
		if option.canDecode() {
			return option.decode
		}
	}
	return httpJsonBodyDecoder{}.decode
}

func getHttpReturner(options ...httpOption) func(Request, int, any) Response {
	for _, option := range options {
		if option.canReturn() {
			return option.Return
		}
	}
	return httpJsonReturner{}.Return
}

func getHttpAuthenticator(options ...httpOption) func(Request) (*User, error) {
	for _, option := range options {
		if option.canAuthenticate() {
			return option.Authenticate
		}
	}
	return httpBaseOption{}.Authenticate
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
	return TemplateResponse{status: http.StatusOK, Reader: reader, err: err}
}

func (r Request) jsonResponse(status int, value any, err error) JsonResponse {
	reader, writer := io.Pipe()
	go func() {
		if err := json.NewEncoder(writer).Encode(value); err != nil {
			r.Log("Could not encode response: %s", err)
		}
		writer.Close()
	}()
	return JsonResponse{status: status, Reader: reader, err: err}
}

func internalServerErrorJsonResponse() Response {
	return JsonResponse{
		status: http.StatusInternalServerError,
		Reader: bytes.NewBuffer([]byte(`{"ok": false}`)),
	}
}

func formError(msg string, t any) map[string]ApiErrors {
	return jsonErrors(ApiErrors{"__form__": []string{translateError(msg, t)}})
}

func jsonErrors(errors ApiErrors) map[string]ApiErrors {
	return map[string]ApiErrors{"errors": errors}
}

func HXRefresh(handler func(Request) Response) func(Request) Response {
	return func(r Request) Response {
		res := handler(r)
		if res.Status() == http.StatusOK {
			r.w.Header().Set("HX-Refresh", "true")
		}

		return res
	}
}

func HXTriggerAfterSwap(handler func(Request) Response, event string) func(Request) Response {
	return func(r Request) Response {
		res := handler(r)
		if res.Status() == http.StatusOK {
			r.w.Header().Set("HX-Trigger-After-Swap", event)
		}

		return res
	}
}

var (
	TEMPLATES      = make(map[string]*template.Template)
	TEMPLATES_LOCK sync.Mutex

	templateFuncs = map[string]any{
		"dict": func(values ...any) (map[string]any, error) {
			if len(values)%2 != 0 {
				return nil, fmt.Errorf("invalid dict call")
			}
			dict := make(map[string]any, len(values)/2)
			for i := 0; i < len(values); i += 2 {
				key, ok := values[i].(string)
				if !ok {
					return nil, fmt.Errorf("dict keys must be strings")
				}
				dict[key] = values[i+1]
			}
			return dict, nil
		},
	}
)

func getTemplate(filename string) *template.Template {
	TEMPLATES_LOCK.Lock()
	defer TEMPLATES_LOCK.Unlock()
	return TEMPLATES[filename]
}

func initTemplates(templateFiles embed.FS, userTemplateFuncs map[string]any) error {
	TEMPLATES_LOCK.Lock()
	defer TEMPLATES_LOCK.Unlock()

	files, err := templateFiles.ReadDir("templates")
	if err != nil {
		return fmt.Errorf("could not read dir: %w", err)
	}

	funcs := MergeMaps(templateFuncs, userTemplateFuncs)
	for _, f := range files {
		name := f.Name()
		tmpl, err := template.New(name).Funcs(funcs).ParseFS(templateFiles, "templates/layout.html", "templates/"+name)
		if err != nil {
			return fmt.Errorf("could not parse template %s: %w", name, err)
		}
		TEMPLATES[name] = tmpl
	}

	return nil
}
