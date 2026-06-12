package capability

import (
	"context"

	"github.com/MortalArena/Musketeers/pkg/policy"
)

type testCommand struct {
	name string
	args map[string]any
}

func (c testCommand) Name() string         { return c.name }
func (c testCommand) Args() map[string]any { return c.args }

type testCapability struct {
	name string
}

func (c *testCapability) Name() string { return c.name }
func (c *testCapability) Execute(_ context.Context, principal policy.Principal, cmd Command) (*Result, error) {
	return &Result{Name: cmd.Name(), Output: map[string]any{"principal": principal.DID, "value": cmd.Args()["value"]}}, nil
}
