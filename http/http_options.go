package http

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	app "github.com/oriolf/simple-app"
)

type httpOption interface {
	canDecode() bool
	decode(Request) (map[string]any, error)

	canReturn() bool
	Return(Request, int, any) Response

	canAuthenticate() bool
	Authenticate(Request) (*app.User, error)
}

type httpBaseOption struct{}

func (o httpBaseOption) canDecode() bool                         { return false }
func (o httpBaseOption) decode(Request) (map[string]any, error)  { return nil, nil }
func (o httpBaseOption) canReturn() bool                         { return false }
func (o httpBaseOption) Return(Request, int, any) Response       { return JsonResponse{} }
func (o httpBaseOption) canAuthenticate() bool                   { return false }
func (o httpBaseOption) Authenticate(Request) (*app.User, error) { return nil, nil }

// Input formats

func FormParams() httpOption { return httpFormDecoder{} }

type httpFormDecoder struct{ httpBaseOption }

func (d httpFormDecoder) canDecode() bool { return true }
func (d httpFormDecoder) Decode(r Request) (map[string]any, error) {
	if err := r.r.ParseForm(); err != nil {
		return nil, err
	}

	params := make(map[string]any)
	for k, lst := range r.r.Form {
		if len(lst) > 0 {
			params[k] = lst[0]
		}
	}

	return params, nil
}

func JsonDecoder() httpOption { return httpJsonBodyDecoder{} }

type httpJsonBodyDecoder struct{ httpBaseOption }

func (d httpJsonBodyDecoder) canDecode() bool { return true }
func (d httpJsonBodyDecoder) decode(r Request) (params map[string]any, err error) {
	decoder := json.NewDecoder(r.r.Body)
	err = decoder.Decode(&params)
	return params, err
}

// Output formats and actions

func JsonReturner() httpOption { return httpJsonReturner{} }

type httpJsonReturner struct {
	httpBaseOption
}

func (o httpJsonReturner) canReturn() bool { return true }
func (o httpJsonReturner) Return(r Request, status int, data any) Response {
	return r.jsonResponse(status, data, nil)
}

func Redirect(url string) httpOption {
	return httpRedirectReturner{URL: url}
}

type httpRedirectReturner struct {
	httpBaseOption
	URL string
}

func (o httpRedirectReturner) canReturn() bool { return true }
func (o httpRedirectReturner) Return(r Request, status int, data any) Response {
	http.Redirect(r.w, r.r, o.URL, http.StatusFound)
	return RedirectResponse{http.StatusFound, bytes.NewBuffer([]byte{})}
}

// Authentication

type httpDefaultAuthenticator struct{ httpBaseOption }

var DefaultAuthentication = httpDefaultAuthenticator{}

func (httpDefaultAuthenticator) canAuthenticate() bool { return true }
func (httpDefaultAuthenticator) Authenticate(r Request) (*app.User, error) {
	c, err := r.r.Cookie("_session")
	if err != nil {
		return nil, fmt.Errorf("could not get cookie: %w", err)
	}

	var userID uint
	err = r.DB.QueryRow("SELECT user_id FROM sessions WHERE id=? AND expires > ?;", c.Value, app.Now()).Scan(&userID)
	if err != nil {
		return nil, fmt.Errorf("session not found: %w", err)
	}

	user, err := app.DBGet(r.DB, app.User{}, userID, struct{}{})
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	return &user, nil
}

// Grouping

type httpGroup struct {
	options []httpOption
}

func Group(options ...httpOption) httpGroup {
	return httpGroup{options}
}
