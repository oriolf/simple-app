package http

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/oriolf/simple-app/db"

	app "github.com/oriolf/simple-app"
	de "github.com/oriolf/simple-app/domain-events"
)

func Add[T app.Adder](seed func() T) func(Request) Response {
	return func(r Request) Response {
		params, err := r.Parameters()
		a := seed()
		if err != nil {
			r.Log("Could not decode data: %s", err)
			return r.JsonGlobalError(http.StatusBadRequest, "Petició mal formada", err)
		}

		if apiErrors := a.Validate(params); apiErrors.NotEmpty() {
			return r.JsonError(http.StatusUnprocessableEntity, apiErrors, nil)
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

		if apiErrors := a.Validate(params); apiErrors.NotEmpty() {
			return r.JsonError(http.StatusUnprocessableEntity, apiErrors, nil)
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
		if apiErrors := a.Validate(params); apiErrors.NotEmpty() {
			return r.JsonError(http.StatusUnprocessableEntity, apiErrors, nil)
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

		field, value, apiErrors := seed.ValidatePatch(key, value)
		if apiErrors.NotEmpty() {
			return r.JsonError(http.StatusUnprocessableEntity, apiErrors, nil)
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

func ExecuteCommand[R de.Command, T de.Entity](commander de.Commander[R, T]) func(Request) Response {
	return func(r Request) Response {
		command := commander.SeedCommand()
		if err := r.TypedParameters(&command); err != nil {
			return r.JsonGlobalError(http.StatusBadRequest, err.Error(), err)
		}

		entity := commander.SeedEntity()
		query := "SELECT * FROM domain_events WHERE entity_id = ? ORDER BY id ASC"
		events, err := db.QueryDB(de.ScanDomainEvent, query, command.EntityID())
		if err != nil {
			return r.JsonGlobalError(http.StatusInternalServerError, err.Error(), err)
		}

		concurrencyMsg := fmt.Sprintf("L'entitat %s té una versió més recent de l'esperada", command.EntityID())
		if len(events) > 0 && app.Last(events).EntityVersion != command.ExpectedVersion() {
			return r.JsonGlobalError(http.StatusConflict, concurrencyMsg, nil)
		}

		de.Hydrate(entity, events)
		f := func(tx *sql.Tx) (err error) {
			event, apiErrors := entity.Execute(tx, command)
			if apiErrors.NotEmpty() {
				return apiErrors
			}

			if err := de.RecordEvent(tx, event); err != nil {
				if strings.Contains(err.Error(), "UNIQUE constraint failed: domain_events.entity_id, domain_events.entity_version") {
					return app.ConcurrencyError
				}
				return err
			}

			return nil
		}
		if err := db.Transaction(f); err != nil {
			var apiErrors app.ApiErrors
			if errors.As(err, &apiErrors) {
				return r.JsonError(http.StatusUnprocessableEntity, apiErrors, nil)
			}
			if errors.Is(err, app.ConcurrencyError) {
				return r.JsonGlobalError(http.StatusConflict, concurrencyMsg, nil)
			}
			return r.JsonGlobalError(http.StatusInternalServerError, err.Error(), err)
		}

		de.InformEventRecorded()
		return r.JsonResponse(map[string]any{"ok": true})
	}
}
