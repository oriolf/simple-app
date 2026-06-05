package entities

import (
	"github.com/oriolf/simple-app/types"
	"github.com/oriolf/simple-app/validators"

	app "github.com/oriolf/simple-app"
	de "github.com/oriolf/simple-app/domain-events"
	vo "github.com/oriolf/simple-app/examples/event-sourcing/value-objects"
)

// TODO define value objects for every field; do not allow to directly create types except by its constructor

// entity, all value objects, cannot create or modify directly, only by its methods, can change its internals
type Member struct {
	ID       types.UUID
	Version  vo.EntityVersion
	Name     vo.Name
	NIF      vo.DNI
	JoinedOn types.Date
	LeftOn   *types.Date
	// iban     *vo.IBAN
}

// TODO go from http handler to event saving; general HTTP handler to execute
// command, that creates a transaction for all the process, etc.

// TODO we need this to ensure DNI uniqueness: https://event-driven.io/en/uniqueness-in-event-sourcing/

type CreateMemberCommand struct {
	expectedVersion vo.EntityVersion
	name            string
	nif             string
	joinedOn        string
}

func (c CreateMemberCommand) SeedEntity() Member { return Member{} }

type MemberCreatedEvent de.DomainEvent

func (m *Member) Create(e CreateMemberCommand) (de.DomainEvent, app.ApiErrors) {
	return de.DomainEvent{}, app.ApiErrors{} // TODO implement
}

func (m *Member) Execute(c app.Command) (de.DomainEvent, app.ApiErrors) {
	switch c := c.(type) {
	case CreateMemberCommand:
		return m.Create(c)
	default:
		return de.DomainEvent{}, validators.NewValidator(nil).Errors()
	}
}

func (m *Member) Hydrate(e de.DomainEvent) {
	switch e.Name {
	}
}
