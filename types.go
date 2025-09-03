package app

import (
	"database/sql"
	"fmt"
	"time"
)

type Date struct {
	Year  uint
	Month time.Month
	Day   uint
}

func (d *Date) MarshalJSON() ([]byte, error) {
	s := fmt.Sprintf(`"%d-%02d-%02d"`, d.Year, d.Month, d.Day)
	return []byte(s), nil
}

type ApiErrors = map[string][]string

type Adder interface {
	GetID() uint
	Add(*sql.Tx) error
	Validate() ApiErrors
}

type Getter interface {
	Get(*sql.DB, uint) error
}

type Option func() error

type Command struct {
	Name     string
	Handler  func()
	Commands []Command
}
