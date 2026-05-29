package http

import (
	"fmt"
	"net/http"

	"github.com/oriolf/simple-app/db"
	"github.com/oriolf/simple-app/types"
	"github.com/oriolf/simple-app/users"
)

type option func(func(Request) Response) func(Request) Response

// Return options

func Redirect(url string) func(func(Request) Response) func(Request) Response {
	return func(handler func(Request) Response) func(Request) Response {
		return func(r Request) Response {
			response := handler(r)
			if response.InternalError() != nil {
				return response
			}

			return newRedirectResponse(url)
		}
	}
}

// Authentication

func UserAuthentication(handler func(Request) Response) func(Request) Response {
	return func(r Request) Response {
		var err error
		r.User, err = authenticate(r)
		if err != nil {
			r.Log("Got an authentication error: %s", err)
			return r.JsonGlobalError(http.StatusUnauthorized, "", err)
		}
		return handler(r)
	}
}

func authenticate(r Request) (*users.User, error) {
	c, err := r.r.Cookie("_session")
	if err != nil {
		return nil, fmt.Errorf("could not get cookie: %w", err)
	}

	var userID uint
	err = db.DB().QueryRow("SELECT user_id FROM sessions WHERE id=? AND expires > ?;", c.Value, types.Now()).Scan(&userID)
	if err != nil {
		return nil, fmt.Errorf("session not found: %w", err)
	}

	user, err := db.Get(users.User{}, userID, struct{}{})
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	return &user, nil
}

// HTMX

func HXTriggerAfterSwap(event string) func(func(Request) Response) func(Request) Response {
	return func(handler func(Request) Response) func(Request) Response {
		return func(r Request) Response {
			res := handler(r)
			if res.Status() == http.StatusOK {
				r.w.Header().Set("HX-Trigger-After-Swap", event)
			}

			return res
		}
	}
}
