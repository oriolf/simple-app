// TODO migrate projects temperatures, borses, monitoring... to validate that
// the design applies to enough use cases, and check the implementation lines
// reduction achieved
package app

import (
	"database/sql"
	"embed"
	"fmt"
	"log"
	"os"
	"strings"

	telegram "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var (
	db  *sql.DB
	bot *telegram.BotAPI

	TELEGRAM_CHAT int64
)

func Init(options ...Option) (err error) {
	for _, opt := range options {
		if err := opt(); err != nil {
			return err
		}
	}
	return nil
}

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

func InitTelegram(token string, chat int64) Option {
	return func() (err error) {
		if bot, err = initTelegram(token, chat); err != nil {
			return fmt.Errorf("could not initialize telegram: %s", err)
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

func initTelegram(token string, chat int64) (*telegram.BotAPI, error) {
	TELEGRAM_CHAT = chat
	return telegram.NewBotAPI(token)
}

func SendTelegram(text string) error {
	msg := telegram.NewMessage(TELEGRAM_CHAT, text)
	if _, err := bot.Send(msg); err != nil {
		return fmt.Errorf("could not send: %w", err)
	}
	return nil
}
