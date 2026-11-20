package db

import (
	"database/sql"

	app "github.com/oriolf/simple-app"
)

type Scanner[T any] interface {
	Scan(*sql.Rows) (T, error)
}

type SQLInserter interface {
	SQLInsert(*sql.Tx) (sql.Result, error)
}

type SQLUpdater interface {
	SQLUpdate(*sql.Tx) error
}

type SQLWhereParamer interface {
	SQLWhereParams() []any
}

type SQLWhereCriteriaParamer[C any] interface {
	SQLWhereCriteriaParams(C) []any
}

type SQLSelecter[C any] interface {
	SelectSQL(C) string
}

type SQLCounter[C any] interface {
	CountSQL(C) string
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
}

type SQLJoinMerger[T any, K app.ID] interface {
	app.IDMerger[T, K]
}

type SQLJoinLister[T, C any, K app.ID] interface {
	SQLLister[T, C]
	SQLJoinMerger[T, K]
}
