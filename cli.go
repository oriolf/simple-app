package app

import (
	"database/sql"
	"fmt"
	"strings"
)

func CLIAdd[T Adder](seed func() T) func([]string) []string {
	return func(args []string) []string {
		a := seed()

		params := parseCliParams(args)

		if errors := a.Validate(params); len(errors) > 0 {
			msgs := []string{"Hi ha errors en els paràmetres:"}
			for k, v := range errors {
				msgs = append(msgs, k+":")
				for _, msg := range v {
					msgs = append(msgs, "    "+msg)
				}
			}
			return msgs
		}

		var id uint
		f := func(tx *sql.Tx) (err error) {
			id, err = a.Add(tx)
			return err
		}
		if err := transaction(db, f); err != nil {
			return []string{fmt.Sprintf("Could not add from cli: %s", err)}
		}

		return []string{fmt.Sprintf("Created user %d", id)}
	}
}

func parseCliParams(args []string) map[string]any {
	m := make(map[string]any)

	i := 0
	for i < len(args) {
		x := args[i]
		if strings.HasPrefix(x, "-") {
			m[strings.TrimPrefix(x, "-")] = args[i+1]
			i++
		}
		i++
	}

	return m
}
