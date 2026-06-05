package domain_events

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/oriolf/simple-app/db"
	"github.com/oriolf/simple-app/types"

	app "github.com/oriolf/simple-app"
)

type DomainEventName string

type EntityVersion uint

type DomainEvent struct {
	ID            uint
	OccurredOn    types.DateTime
	Name          DomainEventName
	EntityID      types.UUID
	EntityVersion EntityVersion
	payload       types.JSON
}

type Projecter interface {
	TableName() string
	Project(*sql.Tx, []DomainEvent) error
}

var newEventRecorded = make(chan struct{}, 100)

// TODO do also subscribers that do not care about the whole entity being
// hydrated, but only care about each event, and are therefore easier to
// implement; examples: project the total fees paid (only care about the
// fee.paid event), or send an email when a user registers (only care about the
// user.registered event)

func InitProjections(projecters ...Projecter) app.Option {
	return func() (err error) {
		projecterEventChannels := []chan struct{}{}
		for _, projecter := range projecters {
			newEvent := make(chan struct{}, 10)
			projecterEventChannels = append(projecterEventChannels, newEvent)
			go project(projecter, newEvent)
		}

		go func() {
			for range newEventRecorded {
				for _, ch := range projecterEventChannels {
					ch <- struct{}{}
				}
			}
		}()

		return nil
	}
}

func RecordEvent(tx *sql.Tx, e DomainEvent) error {
	sql := "INSERT INTO domain_events (occurred_on, name, entity_id, entity_version, payload) VALUES (?, ?, ?, ?, ?);"
	_, err := tx.Exec(sql, e.OccurredOn, e.Name, e.EntityID, e.EntityVersion, e.payload)
	if err != nil {
		return err
	}

	newEventRecorded <- struct{}{}
	return nil
}

func Log(msg string, args ...any) {
	log.Printf("[EVENT] "+msg+"\n", args...)
}

func project(projecter Projecter, newEvent <-chan struct{}) {
	for {
		resultChan := executeProjection(projecter)
		for {
			select {
			case <-newEvent:
				continue
			case result := <-resultChan:
				if result.err != nil {
					Log("Could not execute projection: %s", result.err)
				}
				if result.repeat {
					continue
				}
				break
			}
		}

		select {
		case <-time.After(1 * time.Minute):
			continue
		case <-newEvent:
			continue
		}
	}
}

type projectionResult struct {
	repeat bool
	err    error
}

func executeProjection(projecter Projecter) chan projectionResult {
	ch := make(chan projectionResult)
	go func() {
		repeat, err := executeProjectionAux(projecter)
		ch <- projectionResult{repeat, err}
	}()

	return ch
}

func executeProjectionAux(projecter Projecter) (repeat bool, err error) {
	var lastProcessedEvent uint
	sql := "SELECT last_event FROM projections WHERE name = ?"
	if err := db.DB().QueryRow(sql, projecter.TableName()).Scan(&lastProcessedEvent); err != nil {
		return false, fmt.Errorf("could not get projection's last event: %w", err)
	}
	processToEvent := lastProcessedEvent + 1000

	err = db.Transaction(projectEntities(projecter, lastProcessedEvent, lastProcessedEvent+1000))
	if err != nil {
		return false, fmt.Errorf("could not project entities: %w", err)
	}

	var lastExistingEvent uint
	// TODO handle error when no events present
	sql = "SELECT MAX(id) FROM domain_events"
	if err := db.DB().QueryRow(sql).Scan(&lastExistingEvent); err != nil {
		return false, fmt.Errorf("could not get last existing event: %w", err)
	}

	if lastExistingEvent > processToEvent {
		return true, nil
	}

	return false, nil
}

func projectEntities(projecter Projecter, lastProcessedEvent, processToEvent uint) func(*sql.Tx) error {
	return func(tx *sql.Tx) error {
		sql := "SELECT DISTINCT entity_id FROM domain_events WHERE id > ? AND id <= ?"
		entityIDs, err := db.QueryDB(types.ScanUUID, sql, lastProcessedEvent, processToEvent)
		if err != nil {
			return fmt.Errorf("could not query entity ids: %w", err)
		}

		for _, id := range entityIDs {
			sql := "SELECT * FROM domain_events WHERE entity_id = ? AND id <= ?"
			events, err := db.QueryDB(ScanDomainEvent, sql, id, processToEvent)
			if err != nil {
				return fmt.Errorf("could not query events for entity %s: %w", id, err)
			}

			if err := projecter.Project(tx, events); err != nil {
				return fmt.Errorf("could not project entity %s: %w", id, err)
			}
		}

		return nil
	}
}

func ScanDomainEvent(rows *sql.Rows) (de DomainEvent, err error) {
	return de, rows.Scan(&de.ID, &de.OccurredOn, &de.Name, &de.EntityID, &de.EntityVersion, &de.payload)
}
