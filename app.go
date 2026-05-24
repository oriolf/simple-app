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

	globalErrorTranslations = map[string]string{
		"UNIQUE constraint failed: users.email": "Ja existeix un usuari amb aquest correu electrònic",
	}

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

func InitSQL(migrationFiles embed.FS, dataFolder ...string) Option {
	return func() (err error) {
		if db, err = initSQL(migrationFiles, dataFolder...); err != nil {
			return fmt.Errorf("could not initialize sql: %w", err)
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

func RegisterErrorTranslations(translations map[string]string) Option {
	return func() (err error) {
		for k, v := range translations {
			globalErrorTranslations[k] = v
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
