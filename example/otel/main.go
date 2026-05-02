package main

import (
	"context"
	"errors"
	"log"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

func setupTracer() (*sdktrace.TracerProvider, error) {
	exporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil {
		return nil, err
	}
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	otel.SetTracerProvider(tp)
	return tp, nil
}

func fooErr() error {
	return NewEntityNotFoundError("user", 123)
}

func main() {
	tp, err := setupTracer()
	if err != nil {
		log.Fatal(err)
	}
	defer tp.Shutdown(context.Background()) //nolint:errcheck // best-effort shutdown

	tracer := otel.Tracer("errgen-otel-example")
	ctx, span := tracer.Start(context.Background(), "operation")
	_ = ctx
	defer span.End()

	err = fooErr()
	var e *EntityNotFoundError
	if ok := errors.As(err, &e); ok {
		// Records an exception event on the span with all error fields
		// as searchable attributes; exception.message is set automatically
		// from err.Error().
		span.RecordError(e, trace.WithAttributes(e.Attributes()...))
	}
}
