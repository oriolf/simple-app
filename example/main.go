package main

import (
	"embed"
	"log"
	"time"

	app "github.com/oriolf/simple-app"
)

type User struct {
	ID       uint   `json:"id"`
	Email    string `json:"email"`
	Password string `json:"-"`
	Salt     string `json:"-"`
}

type MembershipFee struct {
	ID       uint      `json:"id"`
	MemberID uint      `json:"-"`
	Year     uint      `json:"year"`
	PaidOn   time.Time `json:"paid_on"`
	Quantity uint      `json:"quantity"` // quantity in EUR cents
}

//go:embed migrations
var migrationFiles embed.FS

func main() {
	if err := app.Init(app.InitSQL(migrationFiles)); err != nil {
		log.Fatalln("Could not initialize app:", err)
	}

	app.Execute(
		app.Command{Name: "http", Handler: http},
		app.Command{
			Name: "user",
			Commands: []app.Command{
				{Name: "add", Handler: app.CLIAdd(MemberFactory)},
			},
		},
	)
}

func http() {
	app.HandleHTTP("GET /ok", app.FixedJsonResponse(map[string]bool{"ok": true}))

	// TODO make them authentication required
	app.HandleHTTP("GET /members", app.HTTPList(Member{}))
	app.HandleHTTP("GET /members/{id}", app.HTTPGet(Member{}))
	app.HandleHTTP("POST /members", app.HTTPAdd(MemberFactory))
	// app.HandleHTTP("PUT /members/{id}", app.HTTPUpdate(MemberFactory))
	// app.HandleHTTP("PATCH /members/{id}", app.HTTPPatch(MemberFactory))

	log.Fatalln(app.ServeHTTP())
}
