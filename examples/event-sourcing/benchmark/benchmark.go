package main

import (
	"database/sql"
	"embed"
	"fmt"
	"log"
	"os/exec"
	"time"

	"github.com/oriolf/simple-app/cli"
	"github.com/oriolf/simple-app/db"
	"github.com/oriolf/simple-app/types"
	"github.com/oriolf/simple-app/users"

	app "github.com/oriolf/simple-app"
)

func main() {
	err := app.Init(db.InitDB(embed.FS{}))
	if err != nil {
		log.Fatalln("Could not initialize app:", err)
	}

	cli.Execute(
		cli.Command{Name: "run", Handler: benchmark},
		cli.Command{Name: "seed", Handler: seedBenchmark},
		cli.Command{Name: "clear", Handler: clearBenchmark},
	)
}

type instance struct {
	OccurredOn  types.DateTime
	ProjectedOn types.DateTime
}

func benchmark(args []string) []string {
	v := cli.NewValidator(args)
	seconds := v.ValidateInt("seconds")
	commandsPerSecond := v.ValidateInt("commandsPerSecond")
	if errors := v.Errors(); errors.NotEmpty() {
		return errors.FormatForCli()
	}

	if err := exec.Command("./login.sh").Run(); err != nil {
		return []string{fmt.Sprintf("No s'ha pogut iniciar sessió: %s", err)}
	}

	totalCommands := seconds * commandsPerSecond
	timeBetweenCommands := time.Second / time.Duration(commandsPerSecond)
	start := time.Now()
	ch := make(chan struct{}, totalCommands)
	for i := 0; i < totalCommands; i++ {
		go executeCommand(ch)
		if i < totalCommands-1 {
			time.Sleep(timeBetweenCommands)
		}
	}

	responses := 0
	for range ch {
		responses++
		if responses == totalCommands {
			break
		}
	}

	result := []string{
		fmt.Sprintf("Executades %d comandes en %ds. Ha tardat %s", totalCommands, seconds, time.Since(start)),
	}

	sql := "SELECT e.occurred_on, m.projected_on FROM domain_events e INNER JOIN members_table m ON e.entity_id = m.id"
	results, err := db.Query(scanFunc, sql)
	if err != nil {
		result = append(result, err.Error())
		return result
	}

	if len(results) != totalCommands {
		result = append(result, fmt.Sprintf("S'esperaven %d files; se n'han trobat %d", totalCommands, len(results)))
	}

	var totalLag time.Duration
	for _, r := range results {
		totalLag += r.ProjectedOn.Time.Sub(r.OccurredOn.Time)
	}

	meanLag := totalLag / time.Duration(totalCommands)
	msg := fmt.Sprintf("La mitjana de retard entre l'esdeveniment i la projecció és %s", meanLag)
	result = append(result, msg)
	return result
}

func executeCommand(ch chan struct{}) {
	// TODO look for response if it has an error and report it
	if err := exec.Command("./command.sh").Run(); err != nil {
		log.Println("Error executant la comanda:", err)
	}
	ch <- struct{}{}
}

func scanFunc(rows *sql.Rows) (i instance, err error) {
	return i, rows.Scan(&i.OccurredOn, &i.ProjectedOn)
}

func seedBenchmark([]string) []string {
	user := users.SuperUserFactory()
	user.Validate(map[string]any{
		"email":    "usuari@example.com",
		"password": "usuariusuari",
	})
	f := func(tx *sql.Tx) (err error) {
		user.Add(tx)
		return nil
	}
	db.Transaction(f)
	return []string{}
}

func clearBenchmark([]string) []string {
	f := func(tx *sql.Tx) (err error) {
		tx.Exec("DELETE FROM domain_events;")
		tx.Exec("DELETE FROM members_table;")
		tx.Exec("UPDATE projections SET last_event = 0;")
		return nil
	}
	db.Transaction(f)
	return []string{}
}
