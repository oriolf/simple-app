package app

import (
	"database/sql"
	"embed"
	"fmt"
	"sort"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// TODO time should be datetime in UTC, not timestamp, that is harder to read.
func initSQL(migrationFiles embed.FS) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", "db.db")
	if err != nil {
		return nil, fmt.Errorf("could not open db: %w", err)
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS migrations (
        id   INTEGER NOT NULL PRIMARY KEY,
        name TEXT NOT NULL,
        time TIMESTAMP NOT NULL
    );`)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("could not create migrations table: %w", err)
	}

	const folder = "migrations"
	files, err := migrationFiles.ReadDir(folder)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("could not read migration files: %w", err)
	}

	sort.Slice(files, func(i, j int) bool { return files[i].Name() < files[j].Name() })
	for _, file := range files {
		b, err := migrationFiles.ReadFile(folder + "/" + file.Name())
		if err != nil {
			db.Close()
			return nil, fmt.Errorf("could not read migration file %s: %w", file.Name(), err)
		}

		err = migrateFile(db, file.Name(), string(b))
		if err != nil {
			db.Close()
			return nil, fmt.Errorf("could not execute migration file %s: %w", file.Name(), err)
		}
	}

	return db, nil
}

func migrateFile(db *sql.DB, filename, contents string) error {
	return transaction(db, func(tx *sql.Tx) error {
		var count int
		err := db.QueryRow("SELECT COUNT(1) FROM migrations WHERE name=?;", filename).Scan(&count)
		if err != nil {
			return fmt.Errorf("could not check if previous migration existed: %w", err)
		}

		if count == 0 {
			if _, err := db.Exec(contents); err != nil {
				return fmt.Errorf("could not execute migration: %w", err)
			}

			if _, err := db.Exec("INSERT INTO migrations (name, time) VALUES (?, ?);", filename, time.Now().Unix()); err != nil {
				return fmt.Errorf("could not set migration as executed: %w", err)
			}
		}

		return nil
	})
}

func transaction(db *sql.DB, f func(*sql.Tx) error) error {
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

func Update[T SQLUpdater](tx *sql.Tx, m T) error {
	if err := m.SQLUpdate(tx); err != nil {
		return fmt.Errorf("could not update: %w", err)
	}
	return nil
}

func Get[T SQLGetter[T]](db *sql.DB, m T, id uint) (T, error) {
	items, err := QueryDB(db, m.Scan, m.SelectSQL()+" WHERE id=?;", id)
	if err != nil {
		return m, fmt.Errorf("could not select: %w", err)
	}
	if len(items) == 0 {
		return m, fmt.Errorf("item not found")
	}
	return items[0], nil
}

func List[T SQLLister[T]](db *sql.DB, m T, paginator Paginator) (items []T, total uint, err error) {
	row := db.QueryRow(m.CountSQL())
	if err := row.Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("could not count: %w", err)
	}

	sql := m.SelectSQL() + m.OrderSQL()
	if paginator != nil {
		sql = sql + "LIMIT ? OFFSET ?;"
		limit, offset := paginator.Limit(), paginator.Offset()
		items, err = QueryDB(db, m.Scan, sql, limit, offset)
	} else {
		items, err = QueryDB(db, m.Scan, sql+";")
	}

	if err != nil {
		return nil, 0, fmt.Errorf("could not select: %w", err)
	}

	return items, total, nil
}
