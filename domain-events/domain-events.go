package domain_events

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/oriolf/simple-app/db"
	"github.com/oriolf/simple-app/types"

	app "github.com/oriolf/simple-app"
)

type DomainEventName string

type EntityVersion uint

func NewEntityVersion() EntityVersion            { return 1 }
func (v EntityVersion) Increment() EntityVersion { return EntityVersion(v + 1) }

type DomainEvent struct {
	ID            uint
	OccurredOn    types.DateTime
	Name          DomainEventName
	EntityID      types.UUID
	EntityVersion EntityVersion
	Payload       types.JSON
}

func NewDomainEvent(
	name DomainEventName,
	command Command,
	payload types.JSON,
) DomainEvent {
	return DomainEvent{
		OccurredOn:    types.Now(),
		Name:          name,
		EntityID:      command.EntityID(),
		EntityVersion: command.ExpectedVersion(),
		Payload:       payload,
	}
}

type Projecter interface {
	TableName() string
	Project(*sql.Tx, []DomainEvent) error
}

type Hydrater interface {
	SetEntityID(types.UUID)
	Hydrate(DomainEvent)
}

type Command interface {
	EntityID() types.UUID
	ExpectedVersion() EntityVersion
}

type Executer interface {
	Execute(*sql.Tx, Command) (DomainEvent, app.ApiErrors)
}

type Entity interface {
	Hydrater
	Executer
}

type Commander[R Command, T Entity] interface {
	SeedCommand() R
	SeedEntity() T
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
	query := "INSERT INTO domain_events (occurred_on, name, entity_id, entity_version, payload) VALUES (?, ?, ?, ?, ?);"
	_, err := tx.Exec(query, e.OccurredOn, e.Name, e.EntityID, e.EntityVersion, e.Payload)
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
	tableName := projecter.TableName()
	var lastProcessedEvent uint
	query := "SELECT last_event FROM projections WHERE name = ?"
	err = db.DB().QueryRow(query, tableName).Scan(&lastProcessedEvent)
	if err != nil && errors.Is(err, sql.ErrNoRows) {
		_, err = db.DB().Exec("INSERT INTO projections (name) VALUES (?);", tableName)
	}

	if err != nil {
		return false, fmt.Errorf("could not get projection's last event: %w", err)
	}
	var lastExistingEvent uint
	query = "SELECT COALESCE(MAX(id), 0) FROM domain_events"
	if err := db.DB().QueryRow(query).Scan(&lastExistingEvent); err != nil {
		return false, fmt.Errorf("could not get last existing event: %w", err)
	}

	processToEvent := lastProcessedEvent + 1000
	if lastExistingEvent < processToEvent {
		processToEvent = lastExistingEvent
	}

	err = db.Transaction(projectEntities(projecter, lastProcessedEvent, processToEvent))
	if err != nil {
		return false, fmt.Errorf("could not project entities: %w", err)
	}

	if lastExistingEvent > processToEvent {
		return true, nil
	}

	return false, nil
}

func projectEntities(projecter Projecter, lastProcessedEvent, processToEvent uint) func(*sql.Tx) error {
	return func(tx *sql.Tx) error {
		query := "SELECT DISTINCT entity_id FROM domain_events WHERE id > ? AND id <= ?"
		entityIDs, err := db.QueryDB(types.ScanUUID, query, lastProcessedEvent, processToEvent)
		if err != nil {
			return fmt.Errorf("could not query entity ids: %w", err)
		}

		for _, id := range entityIDs {
			query := "SELECT * FROM domain_events WHERE entity_id = ? AND id <= ? ORDER BY id ASC"
			events, err := db.QueryDB(ScanDomainEvent, query, id, processToEvent)
			if err != nil {
				return fmt.Errorf("could not query events for entity %s: %w", id, err)
			}

			if err := projecter.Project(tx, events); err != nil {
				return fmt.Errorf("could not project entity %s: %w", id, err)
			}
		}

		query = "UPDATE projections SET last_event = ? WHERE name = ?;"
		_, err = tx.Exec(query, processToEvent, projecter.TableName())
		if err != nil {
			return fmt.Errorf("could not update projection last event: %w", err)
		}

		return nil
	}
}

func ScanDomainEvent(rows *sql.Rows) (de DomainEvent, err error) {
	return de, rows.Scan(&de.ID, &de.OccurredOn, &de.Name, &de.EntityID, &de.EntityVersion, &de.Payload)
}

func Hydrate(h Hydrater, events []DomainEvent) {
	h.SetEntityID(events[0].EntityID)
	for _, e := range events {
		h.Hydrate(e)
	}
}
