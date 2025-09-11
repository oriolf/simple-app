// TODO add automatic created_at, updated_at, deleted_at fields to models, if possible at the database level
package main

import (
	"embed"
	"log"
	"net/http"

	app "github.com/oriolf/simple-app"
)

//go:embed migrations
var migrationFiles embed.FS

//go:embed static
var staticFiles embed.FS

//go:embed templates
var templateFiles embed.FS

func main() {
	if err := app.Init(app.InitSQL(migrationFiles), app.InitTemplates(templateFiles)); err != nil {
		log.Fatalln("Could not initialize app:", err)
	}

	app.Execute(
		app.Command{Name: "http", Handler: httpHandlers},
		app.Command{
			Name: "user",
			Commands: []app.Command{
				{Name: "add", Handler: app.CLIAdd(MemberFactory)},
			},
		},
	)
}

func httpHandlers() {
	// TODO make them authentication required
	app.HandleHTTP("GET /api/members", app.HTTPList(Member{}))
	app.HandleHTTP("GET /api/members/{id}", app.HTTPGet(Member{}))
	app.HandleHTTP("POST /api/members", app.HTTPAdd(MemberFactory))
	app.HandleHTTP("PUT /api/members/{id}", app.HTTPUpdate(MemberFactory))
	app.HandleHTTP("DELETE /api/members/{id}", app.HTTPDelete(Member{}))
	// app.HandleHTTP("PATCH /api/members/{id}", app.HTTPPatch(MemberFactory))

	app.HandleHTTP("GET /index.html", app.HTTPTemplate("index.html"))

	app.HandleHTTP("GET /ok", app.FixedJsonResponse(map[string]bool{"ok": true}))
	http.Handle("/", http.FileServerFS(staticFiles))

	log.Fatalln(app.ServeHTTP())
}
