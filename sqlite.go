package app

import (
	"database/sql"
	"embed"
	"fmt"
	"sort"

	_ "github.com/mattn/go-sqlite3"
)

//go:embed migrations
var simpleMigrations embed.FS

func DB() *sql.DB { return db }

func initSQL(migrationFiles embed.FS, dataFolder ...string) (*sql.DB, error) {
	path := "db.db"
	if dataFolder != nil && len(dataFolder) > 0 && dataFolder[0] != "" {
		path = dataFolder[0] + "/" + path
	}

	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("could not open db: %w", err)
	}

	if err := migrateFiles(db, "simple", simpleMigrations); err != nil {
		db.Close()
		return nil, fmt.Errorf("could not migrate simple files: %w", err)
	}

	if err := migrateFiles(db, "app", migrationFiles); err != nil {
		db.Close()
		return nil, fmt.Errorf("could not migrate app files: %w", err)
	}

	return db, nil
}

func migrateFiles(db *sql.DB, app string, migrationFiles embed.FS) error {
	const folder = "migrations"
	files, err := migrationFiles.ReadDir(folder)
	if err != nil {
		return fmt.Errorf("could not read migration files: %w", err)
	}

	sort.Slice(files, func(i, j int) bool { return files[i].Name() < files[j].Name() })
	for _, file := range files {
		b, err := migrationFiles.ReadFile(folder + "/" + file.Name())
		if err != nil {
			return fmt.Errorf("could not read migration file %s: %w", file.Name(), err)
		}

		err = migrateFile(db, app, file.Name(), string(b))
		if err != nil {
			return fmt.Errorf("could not execute migration file %s: %w", file.Name(), err)
		}
	}

	return nil
}

func migrateFile(db *sql.DB, app, filename, contents string) error {
	return Transaction(db, func(tx *sql.Tx) error {
		var count int
		err := db.QueryRow("SELECT COUNT(1) FROM sqlite_master WHERE name='migrations';").Scan(&count)
		if err != nil {
			return fmt.Errorf("could not check if migrations table exists: %w", err)
		}

		if count > 0 {
			err := db.QueryRow("SELECT COUNT(1) FROM migrations WHERE app=? AND name=?;", app, filename).Scan(&count)
			if err != nil {
				return fmt.Errorf("could not check if previous migration exists: %w", err)
			}
		}

		if count == 0 {
			if _, err := db.Exec(contents); err != nil {
				return fmt.Errorf("could not execute migration: %w", err)
			}

			if _, err := db.Exec("INSERT INTO migrations (app, name, time) VALUES (?, ?, ?);", app, filename, Now()); err != nil {
				return fmt.Errorf("could not set migration as executed: %w", err)
			}
		}

		return nil
	})
}

func Transaction(db *sql.DB, f func(*sql.Tx) error) error {
	tx, err := db.Begin()
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("could not begin transaction: %w", err)
	}

	if err := f(tx); err != nil {
		tx.Rollback()
		return fmt.Errorf("could not execute function: %w", err)
	}

	if err := tx.Commit(); err != nil {
		tx.Rollback()
		return fmt.Errorf("could not commit transaction: %w", err)
	}

	return nil
}

func QueryDB[T any](db *sql.DB, scanFunc func(rows *sql.Rows) (T, error), stmt string, args ...any) ([]T, error) {
	rows, err := db.Query(stmt, args...)
	if err != nil {
		return nil, fmt.Errorf("error executing query: %w", err)
	}
	defer rows.Close()

	res := []T{}
	i := 0
	for rows.Next() {
		x, err := scanFunc(rows)
		if err != nil {
			return nil, fmt.Errorf("error scaning row %d: %w", i, err)
		}

		res = append(res, x)
		i += 1
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("final error in rows: %w", err)
	}

	return res, nil
}

func QueryJoinDB[T SQLJoinMerger[T]](
	db *sql.DB,
	scanFunc func(rows *sql.Rows) (T, error),
	stmt string,
	args ...any,
) ([]T, error) {
	rows, err := db.Query(stmt, args...)
	if err != nil {
		return nil, fmt.Errorf("error executing query: %w", err)
	}
	defer rows.Close()

	res := make(map[uint]T)
	orders := make(map[uint]int)
	rowIndex, modelIndex := 0, 0
	for rows.Next() {
		x, err := scanFunc(rows)
		if err != nil {
			return nil, fmt.Errorf("error scaning row %d: %w", rowIndex, err)
		}

		id := x.GetID()
		previous, ok := res[id]
		if ok {
			res[id] = previous.Merge(x)
		} else {
			res[id] = x
			orders[id] = modelIndex
			modelIndex++
		}

		rowIndex++
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("final error in rows: %w", err)
	}

	var list []T
	for _, m := range res {
		list = append(list, m)
	}

	sort.Slice(list, func(i, j int) bool {
		return orders[list[i].GetID()] < orders[list[j].GetID()]
	})

	return list, nil
}

func DBAdd[T SQLInserter](tx *sql.Tx, m T) (uint, error) {
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

func DBGet[C any, T SQLGetter[T, C]](db *sql.DB, m T, id uint, criteria C) (T, error) {
	items, err := QueryDB(db, m.Scan, m.SelectSQL(criteria)+" WHERE id=?;", id)
	if err != nil {
		return m, fmt.Errorf("could not select: %w", err)
	}
	if len(items) == 0 {
		return m, fmt.Errorf("item not found")
	}
	return items[0], nil
}

func DBGetBy[C any, T SQLGetter[T, C]](db *sql.DB, m T, field string, id any, criteria C) (T, error) {
	items, err := QueryDB(db, m.Scan, m.SelectSQL(criteria)+" WHERE "+field+"=?;", id)
	if err != nil {
		return m, fmt.Errorf("could not select: %w", err)
	}
	if len(items) == 0 {
		return m, fmt.Errorf("item not found")
	}
	return items[0], nil
}

func DBList[C any, T SQLLister[T, C]](db *sql.DB, m T, paginator Paginator, criteria C) (items []T, total uint, err error) {
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
		items, err = QueryDB(db, m.Scan, sql, params...)
	} else {
		items, err = QueryDB(db, m.Scan, sql+";", params...)
	}

	if err != nil {
		return nil, 0, fmt.Errorf("could not select: %w", err)
	}

	return items, total, nil
}

func DBListJoin[C any, T SQLJoinLister[T, C]](db *sql.DB, m T, paginator Paginator, criteria C) (items []T, total uint, err error) {
	row := db.QueryRow(m.CountSQL(criteria))
	if err := row.Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("could not count: %w", err)
	}

	sql := m.SelectSQL(criteria) + m.OrderSQL(criteria)
	if paginator != nil {
		sql = sql + "LIMIT ? OFFSET ?;"
		limit, offset := paginator.Limit(), paginator.Offset()
		items, err = QueryJoinDB(db, m.Scan, sql, limit, offset)
	} else {
		items, err = QueryJoinDB(db, m.Scan, sql+";")
	}

	if err != nil {
		return nil, 0, fmt.Errorf("could not select: %w", err)
	}

	return items, total, nil
}
