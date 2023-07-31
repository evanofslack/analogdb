package logger

import (
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/trace"
)

type tracingHook struct {
	serviceName string
}

func newTracerHook(serviceName string) *tracingHook {
	return &tracingHook{
		serviceName: serviceName,
	}
}

func (t tracingHook) Run(e *zerolog.Event, level zerolog.Level, message string) {

	// grab the context from the logger.
	// the logger caller pass in its current context.
	ctx := e.GetCtx()

	// grab the span and trace id from the context
	spanCtx := trace.SpanContextFromContext(ctx)
	if spanCtx.IsValid() {
		e.Str("span_id", spanCtx.SpanID().String())
		e.Str("trace_id", spanCtx.TraceID().String())
		e.Str("service.name", t.serviceName)
	}
}
