package db

import (
	"database/sql"
	"embed"
	"fmt"
	"sort"

	_ "github.com/mattn/go-sqlite3"
	app "github.com/oriolf/simple-app"
	"github.com/oriolf/simple-app/types"
)

//go:embed migrations
var simpleMigrations embed.FS

var db *sql.DB

func DB() *sql.DB { return db }

func InitDB(migrationFiles embed.FS, dataFolder ...string) app.Option {
	return func() (err error) {
		if db, err = initDB(migrationFiles, dataFolder...); err != nil {
			return fmt.Errorf("could not initialize sql: %w", err)
		}
		return nil
	}
}

func initDB(migrationFiles embed.FS, dataFolder ...string) (*sql.DB, error) {
	path := "db.db"
	if dataFolder != nil && len(dataFolder) > 0 && dataFolder[0] != "" {
		path = dataFolder[0] + "/" + path
	}

	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("could not open db: %w", err)
	}

	if _, err := db.Exec("PRAGMA foreign_keys = ON;"); err != nil {
		return nil, fmt.Errorf("could not set foreign keys enforcement: %w", err)
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

func migrateFiles(db *sql.DB, application string, migrationFiles embed.FS) error {
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

		err = migrateFile(db, application, file.Name(), string(b))
		if err != nil {
			return fmt.Errorf("could not execute migration file %s: %w", file.Name(), err)
		}
	}

	return nil
}

func migrateFile(db *sql.DB, application, filename, contents string) error {
	return transaction(db, func(tx *sql.Tx) error {
		var count int
		err := db.QueryRow("SELECT COUNT(1) FROM sqlite_master WHERE name='migrations';").Scan(&count)
		if err != nil {
			return fmt.Errorf("could not check if migrations table exists: %w", err)
		}

		if count > 0 {
			err := db.QueryRow("SELECT COUNT(1) FROM migrations WHERE app=? AND name=?;", application, filename).Scan(&count)
			if err != nil {
				return fmt.Errorf("could not check if previous migration exists: %w", err)
			}
		}

		if count == 0 {
			if _, err := db.Exec(contents); err != nil {
				return fmt.Errorf("could not execute migration: %w", err)
			}

			if _, err := db.Exec("INSERT INTO migrations (app, name, time) VALUES (?, ?, ?);", application, filename, types.Now()); err != nil {
				return fmt.Errorf("could not set migration as executed: %w", err)
			}
		}

		return nil
	})
}

func Transaction(f func(*sql.Tx) error) error {
	return transaction(db, f)
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

func QueryDB[T any](scanFunc func(rows *sql.Rows) (T, error), stmt string, args ...any) ([]T, error) {
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
