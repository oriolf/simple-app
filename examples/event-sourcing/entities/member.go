package entities

import (
	"database/sql"

	"github.com/oriolf/simple-app/types"
	"github.com/oriolf/simple-app/validators"

	app "github.com/oriolf/simple-app"
	de "github.com/oriolf/simple-app/domain-events"
	vo "github.com/oriolf/simple-app/examples/event-sourcing/value-objects"
)

type Member struct {
	ID       types.UUID
	Version  de.EntityVersion
	Name     vo.Name
	NIF      vo.NIF
	JoinedOn types.Date
	LeftOn   *types.Date
	// iban     *vo.IBAN
}

type CreateMemberCommand struct {
	EntityId        types.UUID       `json:"entity_id"`
	Expectedversion de.EntityVersion `json:"entity_version"`
	Name            string           `json:"name"`
	NIF             string           `json:"nif"`
	JoinedOn        string           `json:"joined_on"`
}

func (c CreateMemberCommand) EntityID() types.UUID              { return c.EntityId }
func (c CreateMemberCommand) ExpectedVersion() de.EntityVersion { return c.Expectedversion }
func (c CreateMemberCommand) SeedEntity() *Member               { return &Member{} }
func (c CreateMemberCommand) SeedCommand() CreateMemberCommand {
	return CreateMemberCommand{EntityId: types.NewUUID(), Expectedversion: de.NewEntityVersion()}
}

var (
	memberCreatedEventName = de.DomainEventName("member.created")
)

func NewMemberCreatedEvent(command CreateMemberCommand) de.DomainEvent {
	payload := types.NewJSON(map[string]any{
		"name":      command.Name,
		"nif":       command.NIF,
		"joined_on": command.JoinedOn,
	})
	return de.NewDomainEvent(memberCreatedEventName, command, payload)
}

func (m *Member) Create(tx *sql.Tx, command CreateMemberCommand) (de.DomainEvent, app.ApiErrors) {
	// TODO ensure NIF uniqueness before creating: https://event-driven.io/en/uniqueness-in-event-sourcing/
	event := NewMemberCreatedEvent(command)
	return event, m.hydrate(event)
}

func (m *Member) Execute(tx *sql.Tx, c de.Command) (de.DomainEvent, app.ApiErrors) {
	switch c := c.(type) {
	case CreateMemberCommand:
		// TODO if version > 1 reject, only can create once
		return m.Create(tx, c)
	default:
		return de.DomainEvent{}, app.NewGlobalApiError("Unrecognized command")
	}
}

func (m *Member) SetEntityID(id types.UUID) {
	m.ID = id
}

func (m *Member) Hydrate(e de.DomainEvent) {
	m.hydrate(e)
}

func (m *Member) hydrate(e de.DomainEvent) app.ApiErrors {
	validator := validators.NewValidator(e.Payload.GetMap())
	switch e.Name {
	case memberCreatedEventName:
		m.Name = vo.NewName(validator, "name")
		m.NIF = vo.NewNIF(validator, "nif")
		m.JoinedOn = vo.NewDate(validator, "joined_on")
	}

	m.Version = m.Version.Increment()
	return validator.Errors()
}
