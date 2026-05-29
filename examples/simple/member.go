package main

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"strings"

	app "github.com/oriolf/simple-app"
	"github.com/oriolf/simple-app/db"
	"github.com/oriolf/simple-app/http"
	"github.com/oriolf/simple-app/types"
	"github.com/oriolf/simple-app/validators"
)

type Member struct {
	ID       uint        `json:"id"`
	Name     string      `json:"name"`
	NIF      string      `json:"nif"`
	JoinedOn types.Date  `json:"joined_on"`
	LeftOn   *types.Date `json:"left_on"`
	IBAN     *string     `json:"iban"`
}

func MemberFactory() *Member { return &Member{} }

// Business methods

func (m *Member) SetID(id uint) { m.ID = id }

func (m *Member) Validate(params map[string]any) app.ApiErrors {
	v := validators.NewValidator(params)
	m.Name = v.ValidateStringNonEmpty("name")
	m.NIF = v.ValidateSpanishDNI("nif")
	m.JoinedOn = v.ValidateDate("joined_on")
	// app.ValidateIBAN("iban", m.IBAN),

	return v.Errors()
}

func (m Member) ValidatePatch(field string, value any) (string, any, app.ApiErrors) {
	var res any
	v := validators.NewValidator(map[string]any{field: value})
	switch field {
	case "name":
		res = v.ValidateStringNonEmpty(field)
	case "nif":
		res = v.ValidateSpanishDNI(field)
	case "joined_on":
		res = v.ValidateDate(field)
	default:
		return field, value, app.ApiErrors{Fields: map[string][]string{field: []string{"Camp desconegut"}}}
	}

	return field, res, v.Errors()
}

func (m Member) Add(tx *sql.Tx) (uint, error) {
	return db.Add(tx, m)
}

func (m Member) Update(tx *sql.Tx) error {
	return m.SQLUpdate(tx)
}

func (m Member) Delete(tx *sql.Tx, id uint) error {
	return m.SQLDelete(tx, id)
}

func (m Member) Patch(tx *sql.Tx, id uint, field string, value any) error {
	return m.SQLPatch(tx, id, field, value)
}

func (m Member) Get(id uint) (Member, error) {
	return db.Get(m, id, memberFilterCriteria{})
}

func (m Member) List(
	paginator app.Paginator,
	criteria memberFilterCriteria,
) (members []Member, total uint, err error) {
	return db.List(m, paginator, criteria)
}

// HTTP methods
type memberFilterCriteria struct {
	search string
}

func (m Member) FilterCriteria(params map[string]any) memberFilterCriteria {
	return memberFilterCriteria{search: params["search"].(string)}
}

// SQL methods
func (m Member) SQLInsert(tx *sql.Tx) (sql.Result, error) {
	return tx.Exec("INSERT INTO members (name, nif, joined_on) VALUES (?, ?, ?);",
		m.Name, m.NIF, m.JoinedOn)
}

func (m Member) SQLUpdate(tx *sql.Tx) error {
	_, err := tx.Exec("UPDATE members SET name=?, nif=?, joined_on=? WHERE id=?;",
		m.Name, m.NIF, m.JoinedOn, m.ID)
	return err
}

func (m Member) SQLDelete(tx *sql.Tx, id uint) error {
	_, err := tx.Exec("DELETE FROM members WHERE id=?;", id)
	return err
}

func (m Member) SQLPatch(tx *sql.Tx, id uint, field string, value any) error {
	sql := fmt.Sprintf("UPDATE members SET %s=? WHERE id=?;", field)
	_, err := tx.Exec(sql, value, id)
	return err
}

func (Member) Scan(rows *sql.Rows) (m Member, err error) {
	return m, rows.Scan(&m.ID, &m.Name, &m.NIF, &m.JoinedOn, &m.LeftOn, &m.IBAN)
}

func (m Member) SelectSQL(criteria memberFilterCriteria) string {
	return "SELECT id, name, nif, joined_on, left_on, iban FROM members " + m.whereSQL(criteria)
}
func (m Member) CountSQL(criteria memberFilterCriteria) string {
	return "SELECT COUNT(1) FROM members" + m.whereSQL(criteria) + ";"
}
func (Member) whereSQL(criteria memberFilterCriteria) string {
	if criteria.search != "" {
		return " WHERE (name LIKE ? OR nif LIKE ?)"
	}
	return ""
}

func (Member) OrderSQL(criteria memberFilterCriteria) string { return "ORDER BY joined_on DESC " }
func (Member) SQLParams(criteria memberFilterCriteria) []any {
	if criteria.search != "" {
		search := "%" + criteria.search + "%"
		return []any{search, search}
	}
	return nil
}

type importMembersParams struct {
	SkipFirstRow   bool   `json:"skip_first_row"`
	NameColumn     uint   `json:"name_column"`
	NIFColumn      uint   `json:"nif_column"`
	JoinedOnColumn uint   `json:"joined_on_column"`
	CSV            string `json:"csv"`
}

func importMembers(r http.Request) http.Response {
	var params importMembersParams
	if err := r.DecodeJsonBody(&params); err != nil {
		r.Log("Could not decode data: %s", err)
		return r.JsonBadRequest("Petició mal formada", err)
	}

	rows, err := csv.NewReader(strings.NewReader(params.CSV)).ReadAll()
	if err != nil {
		r.Log("Could not decode CSV: %s", err)
		return r.JsonBadRequest("CSV mal format", err)
	}

	if params.SkipFirstRow && len(rows) < 2 || !params.SkipFirstRow && len(rows) == 0 {
		return r.JsonBadRequest("El CSV ha de contindre una fila com a mínim", nil)
	}

	if params.NameColumn == params.NIFColumn || params.NameColumn == params.JoinedOnColumn || params.NIFColumn == params.JoinedOnColumn {
		return r.JsonBadRequest("Les columnes indicades han de ser totes diferents", nil)
	}

	columnCount := uint(len(rows[0]))
	if params.NameColumn >= columnCount || params.NIFColumn >= columnCount || params.JoinedOnColumn >= columnCount {
		return r.JsonBadRequest("El CSV no conté suficients columnes", nil)
	}

	if params.SkipFirstRow {
		rows = rows[1:]
	}

	var results []any
	f := func(tx *sql.Tx) (err error) {
		for _, row := range rows {
			values := map[string]any{
				"name":      row[params.NameColumn],
				"nif":       row[params.NIFColumn],
				"joined_on": row[params.JoinedOnColumn],
			}
			var a Member
			if errors := a.Validate(values); errors.NotEmpty() {
				results = append(results, map[string]any{"errors": errors.Fields})
				continue
			}

			id, err := a.Add(tx)
			if err != nil {
				results = append(results, map[string]any{"errors": []string{app.TranslateError(err.Error())}})
				continue
			}

			results = append(results, map[string]any{"id": id})
		}

		return nil
	}
	if err := db.Transaction(f); err != nil {
		return r.JsonInternalError(err.Error(), err)
	}

	return r.JsonResponse(map[string]any{"results": results})
}
