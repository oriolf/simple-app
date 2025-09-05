package main

import (
	"database/sql"

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

func MemberFactory() Member { return Member{} }

// Business methods

func (m Member) Validate() app.ApiErrors {
	return app.Validate(
		app.ValidateStringNonEmpty("name", m.Name),
		app.ValidateSpanishDNI("nif", m.NIF),
		app.ValidateDate("joined_on", m.JoinedOn),
		// app.ValidateIBAN("iban", m.IBAN),
	)
}

func (m Member) Add(tx *sql.Tx) (uint, error) {
	return app.Add(tx, m)
}

func (m Member) Get(db *sql.DB, id uint) (Member, error) {
	return app.Get(db, m, id)
}

func (m Member) List(db *sql.DB, paginator app.Paginator) (members []Member, total uint, err error) {
	return app.List(db, m, paginator)
}

// SQL methods
func (m Member) Insert(tx *sql.Tx) (sql.Result, error) {
	return tx.Exec("INSERT INTO members (name, nif, joined_on) VALUES (?, ?, ?);",
		m.Name, m.NIF, m.JoinedOn.String())
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
