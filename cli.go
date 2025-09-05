package app

import (
	"database/sql"
	"log"
)

func CLIAdd[T Adder](seed func() T) func() {
	return func() {
		a := seed()

		// TODO parse flags, etc.
		f := func(tx *sql.Tx) (err error) {
			_, err = a.Add(tx)
			return err
		}
		if err := transaction(db, f); err != nil {
			log.Println("Could not add from cli:", err)
			return
		}

		return
	}
}
