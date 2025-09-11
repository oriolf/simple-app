package app

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Date struct {
	Year  uint
	Month time.Month
	Day   uint
}

func (d Date) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%s"`, d.String())), nil
}

func (d *Date) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `"`)
	if err := d.FromString(s); err != nil {
		return fmt.Errorf("could not scan date: %w", err)
	}

	return nil
}

func (d Date) Value() (driver.Value, error) {
	return d.String(), nil
}

func (d *Date) Scan(value any) error {
	if value == nil {
		return fmt.Errorf("must receive a non-null string")
	}
	switch value := value.(type) {
	case string:
		return d.FromString(value)
	case []byte:
		return d.FromString(string(value))
	}

	return fmt.Errorf("must receive a string")
}

func (d Date) String() string {
	return fmt.Sprintf("%d-%02d-%02d", d.Year, d.Month, d.Day)
}

func (d *Date) FromString(s string) error {
	_, err := fmt.Sscanf(s, `%d-%d-%d`, &d.Year, &d.Month, &d.Day)
	return err
}

type ApiErrors = map[string][]string

type Validator interface {
	Validate(map[string]any) ApiErrors
	ValidationTranslations() map[string]string
}

type Adder interface {
	Add(*sql.Tx) (uint, error)
	Validator
}

type Updater interface {
	SetID(uint)
	Update(*sql.Tx) error
	Validator
}

type Deleter interface {
	Delete(*sql.Tx, uint) error
}

type Getter[T any] interface {
	Get(*sql.DB, uint) (T, error)
}

type Lister[T any] interface {
	List(*sql.DB, Paginator) ([]T, uint, error)
}

type Paginator interface {
	Limit() uint
	Offset() uint
}

type paginator struct {
	page         uint
	itemsPerPage uint
}

func (p paginator) Limit() uint {
	return p.itemsPerPage
}

func (p paginator) Offset() uint {
	return (p.page - 1) * p.itemsPerPage
}

func NewPaginator(r *http.Request) paginator {
	page, _ := strconv.Atoi(r.FormValue("page"))
	itemsPerPage, _ := strconv.Atoi(r.FormValue("itemsPerPage"))
	if page <= 0 {
		page = 1
	}
	if itemsPerPage <= 0 {
		itemsPerPage = 10
	}
	return paginator{page: uint(page), itemsPerPage: uint(itemsPerPage)}
}

type Scanner[T any] interface {
	Scan(*sql.Rows) (T, error)
}

type SQLInserter interface {
	SQLInsert(*sql.Tx) (sql.Result, error)
}

type SQLUpdater interface {
	SQLUpdate(*sql.Tx) error
}

type SQLSelecter interface {
	SelectSQL() string
}

type SQLCounter interface {
	CountSQL() string
}

type SQLOrderer interface {
	OrderSQL() string
}

type SQLGetter[T any] interface {
	Scanner[T]
	SQLSelecter
}

type SQLLister[T any] interface {
	Scanner[T]
	SQLSelecter
	SQLCounter
	SQLOrderer
}

type Option func() error

type Command struct {
	Name     string
	Handler  func()
	Commands []Command
}
