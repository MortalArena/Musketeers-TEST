package pipeline

import (
	"context"
	"fmt"

	"github.com/MortalArena/Musketeers/pkg/capability"
	"github.com/MortalArena/Musketeers/pkg/policy"
	"github.com/MortalArena/Musketeers/pkg/runtime/observability"
)

type Executor func(ctx context.Context, principal policy.Principal, cmd capability.Command) (*capability.Result, error)

type Middleware interface {
	Handle(ctx context.Context, principal policy.Principal, cmd capability.Command, next Executor) (*capability.Result, error)
}

type MiddlewareFunc func(ctx context.Context, principal policy.Principal, cmd capability.Command, next Executor) (*capability.Result, error)

func (f MiddlewareFunc) Handle(ctx context.Context, principal policy.Principal, cmd capability.Command, next Executor) (*capability.Result, error) {
	return f(ctx, principal, cmd, next)
}

type Pipeline struct {
	middlewares []Middleware
}

func New(middlewares ...Middleware) *Pipeline {
	return &Pipeline{middlewares: append([]Middleware(nil), middlewares...)}
}

func (p *Pipeline) Execute(ctx context.Context, principal policy.Principal, cmd capability.Command, executor Executor) (*capability.Result, error) {
	if cmd == nil {
		return nil, fmt.Errorf("command is nil")
	}
	if executor == nil {
		return nil, fmt.Errorf("executor is nil")
	}
	chain := Executor(executor)
	for i := len(p.middlewares) - 1; i >= 0; i-- {
		middleware := p.middlewares[i]
		next := chain
		chain = func(ctx context.Context, principal policy.Principal, cmd capability.Command) (*capability.Result, error) {
			return middleware.Handle(ctx, principal, cmd, next)
		}
	}
	return chain(ctx, principal, cmd)
}

func AuthorizationMiddleware(engine *policy.Engine) Middleware {
	return MiddlewareFunc(func(ctx context.Context, principal policy.Principal, cmd capability.Command, next Executor) (*capability.Result, error) {
		if engine == nil {
			return next(ctx, principal, cmd)
		}
		resource := policy.Resource{Type: "capability", Action: cmd.Name()}
		if args := cmd.Args(); args != nil {
			if resourceType, ok := args["resource_type"].(string); ok && resourceType != "" {
				resource.Type = resourceType
			}
			if action, ok := args["action"].(string); ok && action != "" {
				resource.Action = action
			}
		}
		result, err := engine.Evaluate(policy.Request{Principal: principal, Resource: resource, Context: cmd.Args()})
		if err != nil {
			return nil, err
		}
		if result.Effect != policy.EffectAllow {
			return nil, fmt.Errorf("capability denied: %s", cmd.Name())
		}
		return next(ctx, principal, cmd)
	})
}

func ApprovalMiddleware(engine *policy.ApprovalEngine) Middleware {
	return MiddlewareFunc(func(ctx context.Context, principal policy.Principal, cmd capability.Command, next Executor) (*capability.Result, error) {
		if engine == nil || cmd.Args()["approval_required"] != true {
			return next(ctx, principal, cmd)
		}
		id := fmt.Sprintf("%s:%s", principal.DID, cmd.Name())
		if err := engine.Request(policy.ApprovalRequest{ID: id, Actor: principal.DID, Action: cmd.Name(), Resource: "capability:" + cmd.Name(), Context: cmd.Args()}); err != nil {
			if _, existsErr := engine.Status(id); existsErr == nil {
				status, _ := engine.Status(id)
				if status.State == policy.ApprovalStatePending {
					return nil, fmt.Errorf("approval required: %s", id)
				}
				if status.State == policy.ApprovalStateDenied {
					return nil, fmt.Errorf("approval denied: %s", id)
				}
			}
			return nil, err
		}
		return nil, fmt.Errorf("approval required: %s", id)
	})
}

func CredentialResolverMiddleware(credentials map[string]string) Middleware {
	return MiddlewareFunc(func(ctx context.Context, principal policy.Principal, cmd capability.Command, next Executor) (*capability.Result, error) {
		required, _ := cmd.Args()["credential"].(string)
		if required == "" {
			return next(ctx, principal, cmd)
		}
		if credentials[required] == "" {
			return nil, fmt.Errorf("credential missing: %s", required)
		}
		return next(ctx, principal, cmd)
	})
}

func AuditMiddleware(audit observability.AuditLog, actor string) Middleware {
	return MiddlewareFunc(func(ctx context.Context, principal policy.Principal, cmd capability.Command, next Executor) (*capability.Result, error) {
		if actor == "" {
			actor = principal.DID
		}
		result, err := next(ctx, principal, cmd)
		state := "success"
		if err != nil {
			state = "error"
		}
		if audit != nil {
			_ = audit.Log(observability.AuditEntry{Actor: actor, Action: "capability." + cmd.Name(), Resource: "capability", Result: state})
		}
		return result, err
	})
}

func TracingMiddleware(tracer observability.Tracer) Middleware {
	return MiddlewareFunc(func(ctx context.Context, principal policy.Principal, cmd capability.Command, next Executor) (*capability.Result, error) {
		if tracer == nil {
			return next(ctx, principal, cmd)
		}
		ctx, span := tracer.StartSpan(ctx, "capability."+cmd.Name())
		defer span.End()
		span.SetAttribute("principal.did", principal.DID)
		result, err := next(ctx, principal, cmd)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(observability.StatusError, err.Error())
		} else {
			span.SetStatus(observability.StatusOK, "ok")
		}
		return result, err
	})
}

func ExecutionMiddleware() Middleware {
	return MiddlewareFunc(func(ctx context.Context, principal policy.Principal, cmd capability.Command, next Executor) (*capability.Result, error) {
		return next(ctx, principal, cmd)
	})
}
