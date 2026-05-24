package main

import (
	"database/sql"
	"embed"
	"fmt"
	"log"
	"strconv"

	app "github.com/oriolf/simple-app"
	"github.com/oriolf/simple-app/cli"
	"github.com/oriolf/simple-app/http"
)

//go:embed migrations
var migrationFiles embed.FS

//go:embed static
var staticFiles embed.FS

//go:embed templates
var templateFiles embed.FS

func main() {
	err := app.Init(
		app.RegisterErrorTranslations(
			map[string]string{"UNIQUE constraint failed: members.nif": "Ja existeix un soci amb aquest DNI"},
		),
		app.InitSQL(migrationFiles),
		http.InitTemplates(templateFiles),
	)
	if err != nil {
		log.Fatalln("Could not initialize app:", err)
	}

	cli.Execute(
		cli.Command{Name: "http", Handler: httpHandlers},
		cli.Command{Name: "seed-e2e", Handler: seedE2E},
		cli.Command{Name: "validate", Commands: []cli.Command{
			{Name: "dni", Handler: validateDNI},
		}},
		cli.Command{Name: "member", Commands: []cli.Command{
			{Name: "add", Handler: cli.Add(MemberFactory)},
		}},
		cli.Command{Name: "user", Commands: []cli.Command{
			{Name: "add", Handler: cli.Add(app.SuperUserFactory)},
		}},
		cli.Command{Name: "types", Commands: []cli.Command{
			{Name: "generate", Handler: app.GenerateTypescriptTypes(app.User{}, app.Session{}, Member{})},
		}},
	)
}

func httpHandlers([]string) []string {
	// API
	http.Handle("GET /api/ok", http.FixedJsonResponse(map[string]bool{"ok": true}))
	http.Handle("POST /api/login", http.Login)

	group := http.Group(http.DefaultAuthentication)
	group.Handle("GET /api/me", http.Me)
	group.Handle("DELETE /api/sessions/{id}", http.DeleteSession)
	group.Handle("GET /api/members", http.List(Member{}))
	group.Handle("GET /api/members/{id}", http.Get(Member{}))
	group.Handle("POST /api/members", http.Add(MemberFactory))
	group.Handle("QUERY /api/members", http.QueryAdd(MemberFactory))
	group.Handle("POST /api/members/import", importMembers)
	group.Handle("PUT /api/members/{id}", http.Update(MemberFactory))
	group.Handle("DELETE /api/members/{id}", http.Delete(Member{}))
	group.Handle("PATCH /api/members/{id}", http.Patch(Member{}))

	// HTML + HTMX
	http.Handle("GET /{$}", http.TemplateList("index.html", Member{}))
	http.Handle("DELETE /members/{id}", http.HXTriggerAfterSwap(http.Delete(Member{}), "members-updated"))
	http.Handle("POST /members", http.HXTriggerAfterSwap(http.Add(MemberFactory), "members-updated"))
	http.Handle("GET /members/{id}/patch-field/{field}", HTTPMemberGetEditField)

	// Static
	http.HandleStatic("GET /static/", staticFiles)
	return []string{http.Serve().Error()}
}

func HTTPMemberGetEditField(r http.Request) http.Response {
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

func seedE2E([]string) []string {
	user := app.SuperUserFactory()
	user.Validate(map[string]any{
		"email":    "usuari@example.com",
		"password": "usuariusuari",
	})
	f := func(tx *sql.Tx) (err error) {
		user.Add(tx)
		return nil
	}
	app.Transaction(app.DB(), f)
	return []string{}
}

func validateDNI(args []string) []string {
	return []string{app.ComputeSpanishDNIControlCharacter(args[0])}
}
