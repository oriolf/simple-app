package app

import (
	"database/sql"
	"embed"
	"fmt"

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

func InitTemplates(templateFiles embed.FS, templateFuncs ...map[string]any) Option {
	return func() error {
		funcs := map[string]any{}
		if templateFuncs != nil {
			funcs = templateFuncs[0]
		}
		if err := initTemplates(templateFiles, funcs); err != nil {
			return fmt.Errorf("could not initialize templates: %w", err)
		}
		return nil
	}
}

func InitTelegram(token string, chat int64) Option {
	return func() (err error) {
		TELEGRAM_CHAT = chat
		if bot, err = telegram.NewBotAPI(token); err != nil {
			return fmt.Errorf("could not initialize telegram: %s", err)
		}
		return nil
	}
}

func SendTelegram(text string) error {
	msg := telegram.NewMessage(TELEGRAM_CHAT, text)
	if _, err := bot.Send(msg); err != nil {
		return fmt.Errorf("could not send: %w", err)
	}
	return nil
}
