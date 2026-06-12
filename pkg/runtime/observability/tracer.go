package observability

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Tracer starts distributed tracing spans.
type Tracer interface {
	StartSpan(ctx context.Context, name string) (context.Context, Span)
}

// Span represents an active trace span.
type Span interface {
	SetAttribute(key string, value any)
	SetStatus(code StatusCode, description string)
	End()
	RecordError(err error)
	AddEvent(name string, attributes map[string]any)
}

type StatusCode int

const (
	StatusUnset StatusCode = 0
	StatusOK    StatusCode = 1
	StatusError StatusCode = 2
)

type OTelTracer struct {
	tracer trace.Tracer
}

func NewOTelTracer(serviceName string) *OTelTracer {
	return &OTelTracer{tracer: otel.Tracer(serviceName)}
}

func (t *OTelTracer) StartSpan(ctx context.Context, name string) (context.Context, Span) {
	ctx, span := t.tracer.Start(ctx, name)
	return ctx, &OTelSpan{span: span}
}

type OTelSpan struct {
	span trace.Span
}

func (s *OTelSpan) SetAttribute(key string, value any) {
	switch v := value.(type) {
	case string:
		s.span.SetAttributes(attribute.String(key, v))
	case int:
		s.span.SetAttributes(attribute.Int(key, v))
	case int64:
		s.span.SetAttributes(attribute.Int64(key, v))
	case float64:
		s.span.SetAttributes(attribute.Float64(key, v))
	case bool:
		s.span.SetAttributes(attribute.Bool(key, v))
	default:
		s.span.SetAttributes(attribute.String(key, fmt.Sprintf("%v", v)))
	}
}

func (s *OTelSpan) SetStatus(code StatusCode, description string) {
	var otelCode codes.Code
	switch code {
	case StatusOK:
		otelCode = codes.Ok
	case StatusError:
		otelCode = codes.Error
	default:
		otelCode = codes.Unset
	}
	s.span.SetStatus(otelCode, description)
}

func (s *OTelSpan) End() {
	s.span.End()
}

func (s *OTelSpan) RecordError(err error) {
	s.span.RecordError(err)
}

func (s *OTelSpan) AddEvent(name string, attributes map[string]any) {
	attrs := make([]attribute.KeyValue, 0, len(attributes))
	for k, v := range attributes {
		switch val := v.(type) {
		case string:
			attrs = append(attrs, attribute.String(k, val))
		case int:
			attrs = append(attrs, attribute.Int(k, val))
		case int64:
			attrs = append(attrs, attribute.Int64(k, val))
		case float64:
			attrs = append(attrs, attribute.Float64(k, val))
		case bool:
			attrs = append(attrs, attribute.Bool(k, val))
		default:
			attrs = append(attrs, attribute.String(k, fmt.Sprintf("%v", val)))
		}
	}
	s.span.AddEvent(name, trace.WithAttributes(attrs...))
}
