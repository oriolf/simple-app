package app

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"
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

type DateTime struct {
	time.Time
}

func Now() DateTime                              { return DateTime{time.Now()} }
func (dt DateTime) Add(d time.Duration) DateTime { return DateTime{dt.Time.Add(d)} }

func (d DateTime) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%s"`, d.String())), nil
}

func (d *DateTime) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `"`)
	if err := d.FromString(s); err != nil {
		return fmt.Errorf("could not scan date time: %w", err)
	}

	return nil
}

func (d DateTime) Value() (driver.Value, error) {
	return d.String(), nil
}

func (d *DateTime) Scan(value any) error {
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

func (d DateTime) String() string {
	return d.Format(time.RFC3339)
}

func (d *DateTime) FromString(s string) (err error) {
	d.Time, err = time.Parse(time.RFC3339, s)
	return err
}

type ApiErrors struct {
	Global []string            `json:"global"`
	Fields map[string][]string `json:"fields"`
}

func NewGlobalApiError(msg string) ApiErrors {
	return ApiErrors{Global: []string{msg}}
}

func (e ApiErrors) Empty() bool {
	return len(e.Global) == 0 && len(e.Fields) == 0
}

func (e ApiErrors) NotEmpty() bool {
	return !e.Empty()
}

func (e ApiErrors) FormatForCli() (msgs []string) {
	msgs = append(msgs, "Global:")
	msgs = e.appendCliSubmessages(msgs, e.Global)
	for k, v := range e.Fields {
		msgs = append(msgs, k+":")
		msgs = e.appendCliSubmessages(msgs, v)
	}
	return msgs
}

func (e ApiErrors) appendCliSubmessages(msgs []string, newMessages []string) []string {
	for _, msg := range newMessages {
		msgs = append(msgs, "    "+msg)
	}
	return msgs
}

type Validator interface {
	Validate(map[string]any) ApiErrors
}

type PatchValidator interface {
	ValidatePatch(string, any) (string, any, ApiErrors)
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

type Patcher interface {
	Patch(*sql.Tx, uint, string, any) error
	PatchValidator
}

type Deleter interface {
	Delete(*sql.Tx, uint) error
}

type Getter[T any] interface {
	Get(*sql.DB, uint) (T, error)
}

type Lister[T, C any] interface {
	List(*sql.DB, Paginator, C) ([]T, uint, error)
	FilterCriteria(*http.Request) C
}

type Paginator interface {
	Limit() uint
	Offset() uint
	Page() uint
	ItemsPerPage() uint
	HasPrevious() bool
	Previous() uint
	HasNext() bool
	Next() uint
	Shown() uint
	Total() uint
	SetTotal(uint)
}

type paginator struct {
	page         uint
	itemsPerPage uint
	total        uint
}

func NewPaginator(r *http.Request) *paginator {
	page, _ := strconv.Atoi(r.FormValue("page"))
	itemsPerPage, _ := strconv.Atoi(r.FormValue("itemsPerPage"))
	if page <= 0 {
		page = 0
	}
	if itemsPerPage <= 0 {
		itemsPerPage = 10
	}
	return &paginator{page: uint(page), itemsPerPage: uint(itemsPerPage)}
}

func (p *paginator) SetTotal(total uint) { p.total = total }

func (p paginator) Limit() uint        { return p.itemsPerPage }
func (p paginator) Offset() uint       { return p.page * p.itemsPerPage }
func (p paginator) Page() uint         { return p.page }
func (p paginator) ItemsPerPage() uint { return p.itemsPerPage }
func (p paginator) HasPrevious() bool  { return p.page > 0 }
func (p paginator) Previous() uint     { return p.page - 1 }
func (p paginator) HasNext() bool      { return p.total > (p.page+1)*p.itemsPerPage }
func (p paginator) Next() uint         { return p.page + 1 }
func (p paginator) Total() uint        { return p.total }
func (p paginator) Shown() uint {
	previous := p.itemsPerPage * p.page
	if previous > p.total {
		return 0
	}
	shown := p.total - previous
	if shown > p.itemsPerPage {
		return p.itemsPerPage
	}
	return shown
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

type SQLParamer[C any] interface {
	SQLParams(C) []any
}

type SQLSelecter[C any] interface {
	SelectSQL(C) string
	SQLParamer[C]
}

type SQLCounter[C any] interface {
	CountSQL(C) string
	SQLParamer[C]
}

type SQLOrderer[C any] interface {
	OrderSQL(C) string
}

type SQLGetter[T, C any] interface {
	Scanner[T]
	SQLSelecter[C]
}

type SQLLister[T, C any] interface {
	Scanner[T]
	SQLSelecter[C]
	SQLCounter[C]
	SQLOrderer[C]
}

type SQLJoinMerger[T any] interface {
	GetID() uint
	Merge(T) T
}

type SQLJoinLister[T, C any] interface {
	SQLLister[T, C]
	SQLJoinMerger[T]
}

type Option func() error

func GenerateTypescriptTypes(models ...any) func([]string) []string {
	return func([]string) []string {
		for _, model := range models {
			generateTypescriptTypes(model)
		}
		return nil
	}
}

func generateTypescriptTypes(model any) {
	t := reflect.TypeOf(model)
	s := "export type " + t.Name() + " = {\n"
	for _, field := range reflect.VisibleFields(t) {
		if tag := field.Tag.Get("json"); tag != "" && tag != "-" {
			s += "  " + tag + ": " + getTypescriptType(field.Type) + ";\n"
		}
	}
	s += "}\n"

	ioutil.WriteFile(t.Name()+".ts", []byte(s), 0644)
}

func getTypescriptType(t reflect.Type) string {
	if InSlice(t.Name(), []string{"int", "uint", "float64"}) {
		return "number"
	}
	if t.Name() == "string" || t.Kind() == reflect.String {
		return "string"
	}
	if t.Kind() == reflect.Slice {
		return getTypescriptType(t.Elem()) + "[]"
	}
	if t.Kind() == reflect.Pointer {
		return getTypescriptType(t.Elem()) + "|null"
	}
	if t.Kind() == reflect.Struct {
		// TODO better define date and date time types, compatible with typescript
		if InSlice(t.Name(), []string{"Date", "DateTime"}) {
			return "string"
		}
		return t.Name()
	}
	return "any"
}
