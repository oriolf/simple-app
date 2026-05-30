package http

import (
	"database/sql"
	"net/http"
	"time"

	app "github.com/oriolf/simple-app"
	"github.com/oriolf/simple-app/db"
	"github.com/oriolf/simple-app/types"
	"github.com/oriolf/simple-app/users"
)

func NewSession(userID uint, r Request) users.Session {
	return users.Session{
		ID:      app.GenerateRandomID(),
		UserID:  userID,
		Time:    types.Now(),
		Expires: types.Now().Add(30 * 24 * time.Hour),
		IP:      r.IP(),
		Agent:   r.Agent(),
	}
}

func Login(r Request) Response {
	var u users.User
	params, err := r.Parameters()
	if err != nil {
		r.Log("Could not decode data: %s", err)
		return r.JsonGlobalError(http.StatusBadRequest, "Petició mal formada", err)
	}

	if errors := u.Validate(params); errors.NotEmpty() {
		return r.JsonError(http.StatusUnprocessableEntity, errors, nil)
	}

	u, err = db.GetBy(u, "email", u.Email, struct{}{})
	if err != nil {
		return r.JsonGlobalError(http.StatusBadRequest, "Correu o contrasenya incorrectes", nil)
	}

	if !u.CorrectPassword(params["password"].(string)) {
		return r.JsonGlobalError(http.StatusBadRequest, "Correu o contrasenya incorrectes", nil)
	}

	// TODO make cookies more secure: https://www.calhoun.io/securing-cookies-in-go
	s := NewSession(u.ID, r)
	f := func(tx *sql.Tx) (err error) {
		_, err = db.Add(tx, s)
		return err
	}
	if err := db.Transaction(f); err != nil {
		return r.JsonGlobalError(http.StatusInternalServerError, "", err)
	}
	c := http.Cookie{
		Name:  "_session",
		Value: s.ID,
	}
	http.SetCookie(r.w, &c)

	return r.JsonResponse(map[string]any{"ok": true})
}

func Me(r Request) Response {
	s := users.Session{UserID: r.User.ID}
	sessions, _, err := db.List(s, nil, struct{}{})
	if err != nil {
		return r.JsonGlobalError(http.StatusInternalServerError, "", err)
	}

	r.User.Sessions = sessions
	return r.JsonResponse(r.User)
}

func DeleteSession(r Request) Response {
	id := r.r.PathValue("id")
	var userID uint
	err := db.DB().QueryRow("SELECT user_id FROM sessions WHERE id=?;", id).Scan(&userID)
	if err != nil {
		return r.JsonGlobalError(http.StatusNotFound, err.Error(), err)
	}

	if userID != r.User.ID {
		return r.JsonGlobalError(http.StatusNotFound, "La sessió no existeix", nil)
	}

	if _, err := db.DB().Exec("DELETE FROM sessions WHERE id=?;", id); err != nil {
		return r.JsonGlobalError(http.StatusInternalServerError, err.Error(), err)
	}

	return r.JsonResponse(map[string]any{"ok": true})
}
