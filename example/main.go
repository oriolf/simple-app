package main

import (
	"embed"
	"log"
	"time"

	app "github.com/oriolf/simple-app"
)

type User struct {
	ID       uint64
	Email    string
	Password string
	Salt     string
}

type Member struct {
	ID       uint64
	Name     string
	NIF      string
	JoinedOn app.Date
	LeftOn   *app.Date
	IBAN     string
}

type MembershipFee struct {
	ID       uint64
	MemberID uint64
	Year     uint
	PaidOn   time.Time
	Quantity uint64 // quantity in EUR cents
}

//go:embed migrations
var migrationFiles embed.FS

func main() {
	if err := app.Init(app.InitSQL(migrationFiles)); err != nil {
		log.Fatalln("Could not initialize app:", err)
	}

	app.Execute(app.HTTPCommand(http), app.CLICommand(cli))
}

func http() {
	// if command http, execute all http code
	app.HandleHTTP("GET /ok", app.FixedJsonResponse(map[string]bool{"ok": true}))

	// TODO make them authentication required
	// app.HandleHTTP("GET /members", app.JsonPaginatedList(Member))
	app.HandleHTTP("POST /members", app.Add(MemberFactory))
	// app.HandleHTTP("PUT /members/:id", app.Update(MemberFactory))
	// app.HandleHTTP("PATCH /members/:id", app.Patch(MemberFactory))

	log.Fatalln(app.ServeHTTP())
}

func cli() {
	// if command cli, execute all cli code
	// app.HandleCli("user add", app.Add(User))
}

func MemberFactory() Member { return Member{} }

func (m Member) GetID() uint64 { return m.ID }
func (m Member) Validate() app.ApiErrors {
	return app.Validate(
		app.ValidateStringNonEmpty("name", m.Name),
		// app.ValidateSpanishNIF("nif", m.NIF),
		// app.ValidateDate("joined_on", m.JoinedOn),
		// app.ValidateIBAN("iban", m.IBAN),
	)
}
func (m Member) Add() error {
	log.Println("Adding member...")
	return nil
}
