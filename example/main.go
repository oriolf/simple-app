package main

import (
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
	ID       uint
	Name     string
	NIF      string
	JoinedOn app.Date
	LeftOn   *app.Date
	IBAN     string
}

type MembershipFee struct {
	ID       uint
	MemberID uint
	Year     uint
	PaidOn   time.Time
	Quantity uint // quantity in EUR cents
}

func main() {
	if err := app.Init(); err != nil {
		log.Fatalln("Could not initialize app:", err)
	}

	// if command http, execute all http code
	app.HandleHTTP("GET /ok", app.FixedJsonResponse(map[string]bool{"ok": true}))
	// app.HandleHTTP("GET /members", app.JsonPaginatedList(Member))

	log.Fatalln(app.ServeHTTP())

	// if command cli, execute all cli code
	// app.HandleCli("user add", app.Create(User))
}
