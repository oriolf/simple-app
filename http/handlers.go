package http

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"

	app "github.com/oriolf/simple-app"
	"github.com/oriolf/simple-app/db"
)

func Add[T app.Adder](seed func() T) func(Request) Response {
	return func(r Request) Response {
		params, err := r.Parameters()
		a := seed()
		if err != nil {
			r.Log("Could not decode data: %s", err)
			return r.JsonGlobalError(http.StatusBadRequest, "Petició mal formada", err)
		}

		if errors := a.Validate(params); errors.NotEmpty() {
			return r.JsonError(http.StatusUnprocessableEntity, errors, nil)
		}

		var id uint
		f := func(tx *sql.Tx) (err error) {
			id, err = a.Add(tx)
			return err
		}
		if err := db.Transaction(f); err != nil {
			return r.JsonGlobalError(http.StatusInternalServerError, err.Error(), err)
		}

		return r.JsonResponse(map[string]any{"id": id})
	}
}

func QueryAdd[T app.Adder](seed func() T) func(Request) Response {
	return func(r Request) Response {
		params, err := r.Parameters()
		a := seed()
		if err != nil {
			r.Log("Could not decode data: %s", err)
			return r.JsonGlobalError(http.StatusBadRequest, "Petició mal formada", err)
		}

		if errors := a.Validate(params); errors.NotEmpty() {
			return r.JsonError(http.StatusUnprocessableEntity, errors, nil)
		}

		return r.JsonResponse(map[string]any{"ok": true})
	}
}

func Update[T app.Updater](seed func() T) func(Request) Response {
	return func(r Request) Response {
		id, err := strconv.Atoi(r.r.PathValue("id"))
		a := seed()
		if err != nil {
			return r.JsonGlobalError(http.StatusBadRequest, err.Error(), err)
		}

		decoder := json.NewDecoder(r.r.Body)
		var params map[string]any
		if err := decoder.Decode(&params); err != nil {
			r.Log("Could not decode data: %s", err)
			return r.JsonGlobalError(http.StatusBadRequest, "Petició mal formada", err)
		}

		a.SetID(uint(id))
		if errors := a.Validate(params); errors.NotEmpty() {
			return r.JsonError(http.StatusUnprocessableEntity, errors, nil)
		}

		if err := db.Transaction(a.Update); err != nil {
			return r.JsonGlobalError(http.StatusInternalServerError, err.Error(), err)
		}

		return r.JsonResponse(map[string]any{})
	}
}

func Patch[T app.Patcher](seed T) func(Request) Response {
	return func(r Request) Response {
		id, err := strconv.Atoi(r.r.PathValue("id"))
		if err != nil {
			return r.JsonGlobalError(http.StatusBadRequest, err.Error(), err)
		}

		decoder := json.NewDecoder(r.r.Body)
		var params map[string]any
		if err := decoder.Decode(&params); err != nil {
			r.Log("Could not decode data: %s", err)
			return r.JsonGlobalError(http.StatusBadRequest, "Petició mal formada", err)
		}

		if len(params) == 0 {
			return r.JsonGlobalError(http.StatusBadRequest, "Cal indicar el camp que es vol actualitzar", nil)
		}

		if len(params) > 1 {
			return r.JsonGlobalError(http.StatusBadRequest, "Només es pot actualitzar un camp per petició", nil)
		}

		var key string
		var value any
		for k, v := range params {
			key = k
			value = v
		}

		field, value, errors := seed.ValidatePatch(key, value)
		if errors.NotEmpty() {
			return r.JsonError(http.StatusUnprocessableEntity, errors, nil)
		}

		f := func(tx *sql.Tx) error {
			return seed.Patch(tx, uint(id), field, value)
		}
		if err := db.Transaction(f); err != nil {
			return r.JsonGlobalError(http.StatusInternalServerError, err.Error(), err)
		}

		return r.JsonResponse(map[string]any{})
	}
}

func Delete[T app.Deleter](seed T) func(Request) Response {
	return func(r Request) Response {
		id, err := strconv.Atoi(r.r.PathValue("id"))
		if err != nil {
			return r.JsonGlobalError(http.StatusBadRequest, err.Error(), err)
		}

		f := func(tx *sql.Tx) (err error) { return seed.Delete(tx, uint(id)) }
		if err := db.Transaction(f); err != nil {
			return r.JsonGlobalError(http.StatusInternalServerError, err.Error(), err)
		}

		return r.JsonResponse(map[string]any{})
	}
}

func Get[T app.Getter[T]](seed T) func(Request) Response {
	return func(r Request) Response {
		id, err := strconv.Atoi(r.r.PathValue("id"))
		if err != nil {
			return r.JsonGlobalError(http.StatusBadRequest, err.Error(), err)
		}

		a, err := seed.Get(uint(id))
		if err != nil {
			return r.JsonGlobalError(http.StatusInternalServerError, err.Error(), err)
		}

		return r.JsonResponse(a)
	}
}

func List[C any, T app.Lister[T, C]](seed T) func(Request) Response {
	return func(r Request) Response {
		items, total, err := seed.List(app.NewPaginator(r.MustParameters()), getFilterCriteria(r, seed))
		if err != nil {
			return r.JsonGlobalError(http.StatusInternalServerError, err.Error(), err)
		}

		return r.JsonResponse(map[string]any{
			"total": total,
			"items": items,
		})
	}
}
