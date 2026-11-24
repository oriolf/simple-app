package app

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"

	telegram "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var (
	bot *telegram.BotAPI

	globalErrorTranslations = map[string]string{
		"UNIQUE constraint failed: users.email": "Ja existeix un usuari amb aquest correu electrònic",
	}

	TELEGRAM_CHAT int64
)

type Option func() error

func Init(options ...Option) (err error) {
	for _, opt := range options {
		if err := opt(); err != nil {
			return err
		}
	}
	return nil
}

func LoadDotEnv() error {
	f, err := os.Open(".env")
	if err != nil {
		return fmt.Errorf("could not read .env file: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.Split(scanner.Text(), "=")
		key := line[0]
		value := line[1]
		if err := os.Setenv(key, value); err != nil {
			return fmt.Errorf("could not set environment variable %s: %w", key, err)
		}
	}

	return nil
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
	if bot == nil {
		return errors.New("uninitialized telegram bot")
	}

	msg := telegram.NewMessage(TELEGRAM_CHAT, text)
	if _, err := bot.Send(msg); err != nil {
		return fmt.Errorf("could not send: %w", err)
	}

	return nil
}
