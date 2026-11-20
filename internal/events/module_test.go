package events

import (
	"context"
	"testing"
)

type testAction struct{}

func (testAction) Name() string              { return "test" }
func (testAction) Run(context.Context) error { return nil }

func TestModuleContractsAreTransportNeutral(t *testing.T) {
	var a Action = testAction{}
	if a.Name() != "test" {
		t.Fatalf("action name = %q", a.Name())
	}
	if err := a.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
}
