package main

import (
	"database/sql"

	app "github.com/oriolf/simple-app"
)

type Member struct {
	ID       uint
	Name     string
	NIF      string
	JoinedOn app.Date
	LeftOn   *app.Date
	IBAN     *string
}

func MemberFactory() *Member { return &Member{} }

// Business methods

func (m *Member) SetID(id uint) { m.ID = id }

func (m *Member) Validate(params map[string]any) app.ApiErrors {
	var errors []app.ApiErrors
	m.Name, errors = app.ValidateStringNonEmpty(errors, params, "name")
	m.NIF, errors = app.ValidateSpanishDNI(errors, params, "nif")
	m.JoinedOn, errors = app.ValidateDate(errors, params, "joined_on")
	// app.ValidateIBAN("iban", m.IBAN),

	return app.Validate(errors...)
}

func (m Member) ValidationTranslations() map[string]string {
	return map[string]string{
		"UNIQUE constraint failed: members.nif": "Ja existeix un soci amb aquest DNI",
	}
}

func (m Member) Add(tx *sql.Tx) (uint, error) {
	return app.Add(tx, m)
}

func (m Member) Update(tx *sql.Tx) error {
	return app.Update(tx, m)
}

func (m Member) Get(db *sql.DB, id uint) (Member, error) {
	return app.Get(db, m, id)
}

func (m Member) List(db *sql.DB, paginator app.Paginator) (members []Member, total uint, err error) {
	return app.List(db, m, paginator)
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

func (Member) Scan(rows *sql.Rows) (m Member, err error) {
	return m, rows.Scan(&m.ID, &m.Name, &m.NIF, &m.JoinedOn, &m.LeftOn, &m.IBAN)
}

func (Member) SelectSQL() string {
	return "SELECT id, name, nif, joined_on, left_on, iban FROM members "
}

func (Member) CountSQL() string {
	return "SELECT COUNT(1) FROM members;"
}

func (Member) OrderSQL() string {
	return "ORDER BY joined_on DESC "
}
