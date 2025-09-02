package app

import (
	"log"
)

func CLIAdd[T Adder](seed func() T) func() {
	return func() {
		a := seed()

		// TODO parse flags, etc.
		if err := a.Add(); err != nil {
			log.Println("Could not add from cli:", err)
			return
		}

		return
	}
}
