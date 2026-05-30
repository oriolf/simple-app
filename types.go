package app

import (
	"database/sql"
	"strconv"
)

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
	Get(uint) (T, error)
}

type Lister[T, C any] interface {
	List(Paginator, C) ([]T, uint, error)
}

type FilterCriterier[C any] interface {
	FilterCriteria(map[string]any) C
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

func NewPaginator(params map[string]any) *paginator {
	page, itemsPerPage := 0, 0
	if pageAny, ok := params["page"]; ok {
		page, _ = strconv.Atoi(pageAny.(string))
	}
	if itemsAny, ok := params["itemsPerPage"]; ok {
		itemsPerPage, _ = strconv.Atoi(itemsAny.(string))
	}
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
