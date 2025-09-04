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
	return []byte(fmt.Sprintf(`"%s"`, d.String())), nil
}

func (d *Date) UnmarshalJSON(b []byte) error {
	if _, err := fmt.Sscanf(string(b), `"%d-%d-%d"`, &d.Year, &d.Month, &d.Day); err != nil {
		return fmt.Errorf("could not scan date: %w", err)
	}

	return nil
}

func (d Date) String() string {
	return fmt.Sprintf("%d-%0d-%0d", d.Year, d.Month, d.Day)
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
