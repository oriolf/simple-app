package app

// TODO Interesting approaches in diversos/temperatures
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

func Execute(commands ...command) {
	var commandNames []string
	for _, c := range commands {
		commandNames = append(commandNames, c.name)
	}
	options := fmt.Sprintf(" Choose one of: %s\n", strings.Join(commandNames, ", "))
	if len(os.Args) < 2 {
		log.Fatalf("Unspecified command." + options)
	}

	for _, c := range commands {
		if os.Args[1] == c.name {
			c.handler()
			return
		}
	}

	log.Fatalf("Unknown command «%s»." + options)
}

func HTTPCommand(f func()) command {
	return command{name: "http", handler: f}
}

func CLICommand(f func()) command {
	return command{name: "cli", handler: f}
}
