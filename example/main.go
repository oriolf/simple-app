package main

import (
	"embed"
	"fmt"
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
		app.Command{Name: "member", Commands: []app.Command{
			{Name: "add", Handler: app.CLIAdd(MemberFactory)},
		}},
		app.Command{Name: "user", Commands: []app.Command{
			{Name: "add", Handler: app.CLIAddSuperUser},
		}},
		app.Command{Name: "types", Commands: []app.Command{
			{Name: "generate", Handler: app.GenerateTypescriptTypes(app.User{}, app.Session{}, Member{})},
		}},
	)
}

func httpHandlers([]string) []string {
	// API
	app.HandleHTTP("GET /api/ok", app.FixedJsonResponse(map[string]bool{"ok": true}))
	app.HandleHTTP("POST /api/login", app.Login)

	group := app.HTTPGroup(app.DefaultAuthentication)
	group.HandleHTTP("GET /api/me", app.Me)
	group.HandleHTTP("DELETE /api/sessions/{id}", app.DeleteSession)
	group.HandleHTTP("GET /api/members", app.HTTPList(Member{}))
	group.HandleHTTP("GET /api/members/{id}", app.HTTPGet(Member{}))
	group.HandleHTTP("POST /api/members", app.HTTPAdd(MemberFactory))
	group.HandleHTTP("QUERY /api/members", app.HTTPQueryAdd(MemberFactory))
	group.HandleHTTP("POST /api/members/import", importMembers)
	group.HandleHTTP("PUT /api/members/{id}", app.HTTPUpdate(MemberFactory))
	group.HandleHTTP("DELETE /api/members/{id}", app.HTTPDelete(Member{}))
	group.HandleHTTP("PATCH /api/members/{id}", app.HTTPPatch(Member{}))

	// HTML + HTMX
	app.HandleHTTP("GET /{$}", app.HTTPTemplateList("index.html", Member{}))
	app.HandleHTTP("DELETE /members/{id}", app.HXTriggerAfterSwap(app.HTTPDelete(Member{}), "members-updated"))
	app.HandleHTTP("POST /members", app.HXTriggerAfterSwap(app.HTTPAdd(MemberFactory), "members-updated"))
	app.HandleHTTP("GET /members/{id}/patch-field/{field}", HTTPMemberGetEditField)

	// Static
	http.Handle("GET /static/", http.FileServerFS(staticFiles))
	return []string{app.ServeHTTP().Error()}
}

func HTTPMemberGetEditField(r app.Request) app.Response {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		return r.TemplateError(err)
	}

	field := r.PathValue("field")
	if !app.InSlice(field, []string{"name", "nif", "joined_on"}) {
		return r.TemplateError(fmt.Errorf("Unknown field"))
	}

	member, err := app.DBGet(r.DB, Member{}, uint(id), memberFilterCriteria{})
	if err != nil {
		return r.TemplateError(err)
	}

	return r.TemplateResponse("member_edit_field.html", map[string]any{"field": field, "member": member}, nil)
}
