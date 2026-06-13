package cli

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"reflect"

	app "github.com/oriolf/simple-app"
	"github.com/oriolf/simple-app/db"
)

func Add[T app.Adder](seed func() T) func([]string) []string {
	return func(args []string) []string {
		a := seed()

		params, err := parseCliParams(args)
		if err != nil {
			return []string{err.Error()}
		}

		if errors := a.Validate(params); errors.NotEmpty() {
			return append([]string{"There are errors in the parameters:"}, errors.FormatForCli()...)
		}

		var id uint
		f := func(tx *sql.Tx) (err error) {
			id, err = a.Add(tx)
			return err
		}
		if err := db.Transaction(f); err != nil {
			return []string{fmt.Sprintf("Could not add from cli: %s", app.TranslateError(err.Error()))}
		}

		return []string{fmt.Sprintf("Created entity with ID %d", id)}
	}
}

func GenerateTypescriptTypes(models ...any) func([]string) []string {
	return func([]string) []string {
		for _, model := range models {
			generateTypescriptTypes(model)
		}
		return nil
	}
}

func generateTypescriptTypes(model any) {
	t := reflect.TypeOf(model)
	s := "export type " + t.Name() + " = {\n"
	for _, field := range reflect.VisibleFields(t) {
		if tag := field.Tag.Get("json"); tag != "" && tag != "-" {
			s += "  " + tag + ": " + getTypescriptType(field.Type) + ";\n"
		}
	}
	s += "}\n"

	ioutil.WriteFile(t.Name()+".ts", []byte(s), 0644)
}

func getTypescriptType(t reflect.Type) string {
	if app.InSlice(t.Name(), []string{"int", "uint", "float64"}) {
		return "number"
	}
	if t.Name() == "string" || t.Kind() == reflect.String {
		return "string"
	}
	if t.Kind() == reflect.Slice {
		return getTypescriptType(t.Elem()) + "[]"
	}
	if t.Kind() == reflect.Pointer {
		return getTypescriptType(t.Elem()) + "|null"
	}
	if t.Kind() == reflect.Struct {
		if app.InSlice(t.Name(), []string{"Date", "DateTime"}) {
			return "string"
		}
		return t.Name()
	}
	return "any"
}
