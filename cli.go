package app

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
)

func CLIAdd[T Adder](seed func() T) func([]string) []string {
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
		if err := Transaction(db, f); err != nil {
			return []string{fmt.Sprintf("Could not add from cli: %s", TranslateError(err.Error()))}
		}

		return []string{fmt.Sprintf("Created entity with ID %d", id)}
	}
}

func Execute(commands ...Command) {
	execute(os.Args[1:], commands...)
}

func execute(args []string, commands ...Command) {
	if len(commands) == 1 && len(args) == 0 && commands[0].Name == "" {
		executeHandler(nil, commands[0])
		return
	}

	var commandNames []string
	for _, c := range commands {
		commandNames = append(commandNames, c.Name)
	}
	options := fmt.Sprintf(" Choose one of: %s\n", strings.Join(commandNames, ", "))
	if len(args) < 1 {
		log.Fatalln("Unspecified command." + options)
	}

	for _, c := range commands {
		if args[0] == c.Name {
			if c.Handler != nil {
				executeHandler(args[1:], c)
				return
			} else {
				execute(args[1:], c.Commands...)
				return
			}
		}
	}

	log.Fatalf("Unknown command «%s»."+options, args[0])
}

func executeHandler(args []string, c Command) {
	for _, msg := range c.Handler(args) {
		fmt.Println(msg)
	}
}

func parseCliParams(args []string) (map[string]any, error) {
	m := make(map[string]any)

	i := 0
	for i < len(args) {
		x := args[i]
		if strings.HasPrefix(x, "-") {
			param := strings.TrimPrefix(x, "-")
			if len(args) <= i {
				return m, fmt.Errorf("Missing value for param «%s»", param)
			}
			m[param] = args[i+1]
			i++
		}
		i++
	}

	return m, nil
}
