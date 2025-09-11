// TODO static files from go-sqlite-http-template
// TODO template files from borses.git
package app

import (
	"database/sql"
	"embed"
	"fmt"
	"log"
	"os"
	"strings"
)

var (
	db *sql.DB
)

func Init(options ...Option) (err error) {
	for _, opt := range options {
		if err := opt(); err != nil {
			return err
		}
	}
	return nil
}

// TODO could also init Telegram, Redis...
func InitSQL(migrationFiles embed.FS) Option {
	return func() (err error) {
		if db, err = initSQL(migrationFiles); err != nil {
			return fmt.Errorf("could not initialize sql: %w", err)
		}
		return nil
	}
}

func InitTemplates(templateFiles embed.FS) Option {
	return func() error {
		if err := initTemplates(templateFiles); err != nil {
			return fmt.Errorf("could not initialize templates: %w", err)
		}
		return nil
	}
}

func Execute(commands ...Command) {
	execute(os.Args[1:], commands...)
}

func execute(args []string, commands ...Command) {
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
				c.Handler()
				return
			} else {
				execute(args[1:], c.Commands...)
				return
			}
		}
	}

	log.Fatalf("Unknown command «%s»."+options, args[0])
}
