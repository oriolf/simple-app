package main

import (
	"embed"
	"log"
	"net/http"
	"strconv"

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
		app.Command{Name: "user", Commands: []app.Command{
			{Name: "add", Handler: app.CLIAdd(MemberFactory)},
		}},
	)
}

func httpHandlers() {
	// API
	app.HandleHTTP("GET /api/ok", app.FixedJsonResponse(map[string]bool{"ok": true}))

	// TODO make them authentication required
	app.HandleHTTP("GET /api/members", app.HTTPList(Member{}))
	app.HandleHTTP("GET /api/members/{id}", app.HTTPGet(Member{}))
	app.HandleHTTP("POST /api/members", app.HTTPAdd(MemberFactory))
	app.HandleHTTP("PUT /api/members/{id}", app.HTTPUpdate(MemberFactory))
	app.HandleHTTP("DELETE /api/members/{id}", app.HTTPDelete(Member{}))
	app.HandleHTTP("PATCH /api/members/{id}", app.HTTPPatch(Member{}))

	// HTML + HTMX
	app.HandleHTTP("GET /{$}", app.HTTPTemplateList("index.html", Member{}))
	app.HandleHTTP("DELETE /members/{id}", app.HXTriggerAfterSwap(app.HTTPDelete(Member{}), "members-updated"))
	app.HandleHTTP("POST /members", app.HXTriggerAfterSwap(app.HTTPAdd(MemberFactory), "members-updated"))
	app.HandleHTTP("GET /members/{id}/patch-field/{field}", HTTPMemberGetEditField)

	// Static
	http.Handle("/static/", http.FileServerFS(staticFiles))
	log.Fatalln(app.ServeHTTP())
}

func HTTPMemberGetEditField(r app.Request) app.Response {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		return r.TemplateError()
	}

	field := r.PathValue("field")
	if !app.InSlice(field, []string{"name", "nif", "joined_on"}) {
		return r.TemplateError()
	}

	member, err := app.DBGet(r.DB, Member{}, uint(id))
	if err != nil {
		return r.TemplateError()
	}

	return r.TemplateResponse("member_edit_field.html", map[string]any{"field": field, "member": member})
}
