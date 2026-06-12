package capability

import (
	"context"
	"fmt"

	"github.com/MortalArena/Musketeers/pkg/policy"
)

type Capability interface {
	Name() string
	Execute(ctx context.Context, principal policy.Principal, cmd Command) (*Result, error)
}

type Command interface {
	Name() string
	Args() map[string]any
}

type Result struct {
	Name   string         `json:"name"`
	Output map[string]any `json:"output,omitempty"`
	Error  string         `json:"error,omitempty"`
}

func NewResult(name string, output map[string]any) *Result {
	if output == nil {
		output = map[string]any{}
	}
	return &Result{Name: name, Output: output}
}

func NewErrorResult(name string, err error) *Result {
	if err == nil {
		err = fmt.Errorf("unknown error")
	}
	return &Result{Name: name, Error: err.Error()}
}
