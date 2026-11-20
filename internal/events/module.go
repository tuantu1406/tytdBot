// Package events contains event-agnostic orchestration contracts.
//
// Concrete event implementations stay in the root package for now because
// they need Player/EventPlayer internals. The package deliberately contains
// only transport-neutral types so migrating an event cannot create an import
// cycle or change its wire behavior.
package events

import "context"

// Action is one idempotent event operation. Implementations must not retry
// purchases or item-consuming actions unless the module explicitly authorizes
// them and the server response proves the previous attempt failed.
type Action interface {
	Name() string
	Run(context.Context) error
}

// Module describes the lifecycle surface every event adapter will expose.
// ActivityType is the server activity type, not the per-account activity ID.
type Module interface {
	Name() string
	ActivityType() int32
	Actions() []Action
}

// ActionResult makes orchestration logs stable without exposing credentials or
// raw packet data. It is intentionally small; event-specific details remain in
// the module's own typed state.
type ActionResult struct {
	Name    string
	Success bool
	Retry   bool
}
