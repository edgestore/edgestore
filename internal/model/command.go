package model

import "context"

// Command encapsulates the data to mutate an aggregate.
type Command interface {
	CommandID() ID
	CommandTenantID() ID
}

// CommandModel provides an embeddable struct that implements Command.
type CommandModel struct {
	// ID contains the aggregate id.
	ID ID `json:"id"`

	// TenantID is the of the owner of an event.
	TenantID ID `json:"tenant_id"`
}

// CommandID implements the Command interface; returns the aggregate id
func (m *CommandModel) CommandID() ID {
	return m.ID
}

// CommandTenantID implements the Command interface; returns the username
func (m *CommandModel) CommandTenantID() ID {
	return m.TenantID
}

// CommandHandler consumes a command and emits Events
type CommandHandler interface {
	// Apply applies a command to an aggregate to generate a new set of events
	Apply(ctx context.Context, command Command) ([]Event, error)
}
