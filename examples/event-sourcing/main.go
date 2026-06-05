package main

import (
	"embed"
	"log"

	"github.com/oriolf/simple-app/cli"
	"github.com/oriolf/simple-app/db"
	"github.com/oriolf/simple-app/examples/event-sourcing/entities"
	"github.com/oriolf/simple-app/examples/event-sourcing/projections"
	"github.com/oriolf/simple-app/http"
	"github.com/oriolf/simple-app/users"

	app "github.com/oriolf/simple-app"
	de "github.com/oriolf/simple-app/domain-events"
)

//go:embed migrations
var migrationFiles embed.FS

func main() {
	err := app.Init(
		db.InitDB(migrationFiles),
		de.InitProjections(projections.MemberTableProjecter),
	)
	if err != nil {
		log.Fatalln("Could not initialize app:", err)
	}

	cli.Execute(
		cli.Command{Name: "http", Handler: httpHandlers},
		cli.Command{Name: "user", Commands: []cli.Command{
			{Name: "add", Handler: cli.Add(users.SuperUserFactory)},
		}},
	)
}

func httpHandlers([]string) []string {
	// API
	http.Handle("GET /api/ok", http.FixedJsonResponse(map[string]bool{"ok": true}))
	http.Handle("POST /api/login", http.Login)

	group := http.Group(http.UserAuthentication)
	group.Handle("GET /api/me", http.Me)
	group.Handle("DELETE /api/sessions/{id}", http.DeleteSession)
	group.Handle("GET /api/members", http.List(projections.MemberTable{}))
	group.Handle("POST /api/members", http.ExecuteCommand(entities.CreateMemberCommand{}))

	return []string{http.Serve().Error()}
}
