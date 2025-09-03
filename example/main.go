package main

import (
	"database/sql"
	"embed"
	"fmt"
	"log"
	"time"

	app "github.com/oriolf/simple-app"
)

type User struct {
	ID       uint
	Email    string
	Password string
	Salt     string
}

type Member struct {
	ID       uint      `json:"id"`
	Name     string    `json:"name"`
	NIF      string    `json:"nif"`
	JoinedOn app.Date  `json:"joined_on"`
	LeftOn   *app.Date `json:"left_on"`
	IBAN     string    `json:"iban"`
}

type MembershipFee struct {
	ID       uint
	MemberID uint
	Year     uint
	PaidOn   time.Time
	Quantity uint // quantity in EUR cents
}

//go:embed migrations
var migrationFiles embed.FS

func main() {
	if err := app.Init(app.InitSQL(migrationFiles)); err != nil {
		log.Fatalln("Could not initialize app:", err)
	}

	app.Execute(
		app.Command{
			Name:    "http",
			Handler: http,
		},
		app.Command{
			Name: "user",
			Commands: []app.Command{{
				Name:    "add",
				Handler: app.CLIAdd(MemberFactory),
			}},
		},
	)
}

func http() {
	app.HandleHTTP("GET /ok", app.FixedJsonResponse(map[string]bool{"ok": true}))

	// TODO make them authentication required
	// app.HandleHTTP("GET /members", app.HTTPList(Member))
	app.HandleHTTP("GET /members/{id}", app.HTTPGet(MemberFactory))
	app.HandleHTTP("POST /members", app.HTTPAdd(MemberFactory))
	// app.HandleHTTP("PUT /members/:id", app.HTTPUpdate(MemberFactory))
	// app.HandleHTTP("PATCH /members/:id", app.HTTPPatch(MemberFactory))

	log.Fatalln(app.ServeHTTP())
}

func cli() {
	// app.HandleCli("user add", app.Add(User))
}

func MemberFactory() *Member { return &Member{} }

func (m Member) GetID() uint { return m.ID }
func (m Member) Validate() app.ApiErrors {
	return app.Validate(
		app.ValidateStringNonEmpty("name", m.Name),
		// app.ValidateSpanishNIF("nif", m.NIF),
		// app.ValidateDate("joined_on", m.JoinedOn),
		// app.ValidateIBAN("iban", m.IBAN),
	)
}
func (m *Member) Add(tx *sql.Tx) error {
	res, err := tx.Exec("INSERT INTO members (name) VALUES (?);", m.Name)
	if err != nil {
		return fmt.Errorf("could not insert: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("could not get last id: %w", err)
	}
	m.ID = uint(id)
	return nil
}
func (m *Member) Get(db *sql.DB, id uint) error {
	row := db.QueryRow("SELECT id, name FROM members WHERE id=?;", id)
	if err := row.Scan(&m.ID, &m.Name); err != nil {
		return fmt.Errorf("could not select: %w", err)
	}
	return nil
}
