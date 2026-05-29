package db

import (
	"database/sql"
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
