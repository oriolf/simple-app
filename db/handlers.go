package db

import (
	"database/sql"
	"fmt"

	app "github.com/oriolf/simple-app"
)

func Add[T SQLInserter](tx *sql.Tx, m T) (uint, error) {
	res, err := m.SQLInsert(tx)
	if err != nil {
		return 0, fmt.Errorf("could not insert: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("could not get last id: %w", err)
	}
	return uint(id), nil
}

func Get[C any, T SQLGetter[T, C]](m T, id uint, criteria C) (T, error) {
	items, err := QueryDB(m.Scan, m.SelectSQL(criteria)+" WHERE id=?;", id)
	if err != nil {
		return m, fmt.Errorf("could not select: %w", err)
	}
	if len(items) == 0 {
		return m, fmt.Errorf("item not found")
	}
	return items[0], nil
}

func GetBy[C any, T SQLGetter[T, C]](m T, field string, id any, criteria C) (T, error) {
	items, err := QueryDB(m.Scan, m.SelectSQL(criteria)+" WHERE "+field+"=?;", id)
	if err != nil {
		return m, fmt.Errorf("could not select: %w", err)
	}
	if len(items) == 0 {
		return m, fmt.Errorf("item not found")
	}
	return items[0], nil
}

func List[C any, T SQLLister[T, C]](m T, paginator app.Paginator, criteria C) (items []T, total uint, err error) {
	params := m.SQLParams(criteria)
	row := db.QueryRow(m.CountSQL(criteria), params...)
	if err := row.Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("could not count: %w", err)
	}

	sql := m.SelectSQL(criteria) + m.OrderSQL(criteria)
	if paginator != nil {
		sql = sql + "LIMIT ? OFFSET ?;"
		limit, offset := paginator.Limit(), paginator.Offset()
		params = append(params, limit, offset)
		items, err = QueryDB(m.Scan, sql, params...)
	} else {
		items, err = QueryDB(m.Scan, sql+";", params...)
	}

	if err != nil {
		return nil, 0, fmt.Errorf("could not select: %w", err)
	}

	return items, total, nil
}

func ListJoin[C any, T SQLJoinLister[T, C]](m T, paginator app.Paginator, criteria C) (items []T, total uint, err error) {
	row := db.QueryRow(m.CountSQL(criteria))
	if err := row.Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("could not count: %w", err)
	}

	sql := m.SelectSQL(criteria) + m.OrderSQL(criteria)
	if paginator != nil {
		sql = sql + "LIMIT ? OFFSET ?;"
		limit, offset := paginator.Limit(), paginator.Offset()
		items, err = QueryJoinDB(m.Scan, sql, limit, offset)
	} else {
		items, err = QueryJoinDB(m.Scan, sql+";")
	}

	if err != nil {
		return nil, 0, fmt.Errorf("could not select: %w", err)
	}

	return items, total, nil
}
