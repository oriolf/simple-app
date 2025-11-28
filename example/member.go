package main

import (
	"database/sql"
	"fmt"

	app "github.com/oriolf/simple-app"
)

type Member struct {
	ID       uint      `json:"id"`
	Name     string    `json:"name"`
	NIF      string    `json:"nif"`
	JoinedOn app.Date  `json:"joined_on"`
	LeftOn   *app.Date `json:"left_on"`
	IBAN     *string   `json:"iban"`
}

func MemberFactory() *Member { return &Member{} }

// Business methods

func (m *Member) SetID(id uint) { m.ID = id }

func (m *Member) Validate(params map[string]any) app.ApiErrors {
	v := app.NewValidator(params)
	m.Name = v.ValidateStringNonEmpty("name")
	m.NIF = v.ValidateSpanishDNI("nif")
	m.JoinedOn = v.ValidateDate("joined_on")
	// app.ValidateIBAN("iban", m.IBAN),

	return v.Errors()
}

func (m Member) ValidatePatch(field string, value any) (string, any, app.ApiErrors) {
	var res any
	v := app.NewValidator(map[string]any{field: value})
	switch field {
	case "name":
		res = v.ValidateStringNonEmpty(field)
	case "nif":
		res = v.ValidateSpanishDNI(field)
	case "joined_on":
		res = v.ValidateDate(field)
	default:
		return field, value, app.ApiErrors{field: []string{"Camp desconegut"}}
	}

	return field, res, v.Errors()
}

func (m Member) ValidationTranslations() map[string]string {
	return map[string]string{
		"UNIQUE constraint failed: members.nif": "Ja existeix un soci amb aquest DNI",
	}
}

func (m Member) Add(tx *sql.Tx) (uint, error) {
	return app.DBAdd(tx, m)
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

func (m Member) Get(db *sql.DB, id uint) (Member, error) {
	return app.DBGet(db, m, id)
}

func (m Member) List(db *sql.DB, paginator app.Paginator) (members []Member, total uint, err error) {
	return app.DBList(db, m, paginator)
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

func (Member) SelectSQL() string {
	return "SELECT id, name, nif, joined_on, left_on, iban FROM members "
}
func (Member) CountSQL() string { return "SELECT COUNT(1) FROM members;" }
func (Member) OrderSQL() string { return "ORDER BY joined_on DESC " }
func (Member) SQLParams() []any { return nil }
