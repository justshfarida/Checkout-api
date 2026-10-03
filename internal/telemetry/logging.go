package telemetry

import (
	"context"
	"errors"
	"log/slog"
	"os"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/trace"
)

func configureLogging(provider *sdklog.LoggerProvider) {
	jsonHandler := slog.NewJSONHandler(os.Stdout, nil).WithAttrs(
		[]slog.Attr{
			slog.String("service", os.Getenv("OTEL_SERVICE_NAME")),
		},
	)

	otelHandler := otelslog.NewHandler(
		"checkout-api",
		otelslog.WithLoggerProvider(provider),
	)

	slog.SetDefault(slog.New(&dualHandler{
		json: jsonHandler,
		otel: otelHandler,
	}))
}

type dualHandler struct {
	json slog.Handler
	otel slog.Handler
}

func (h *dualHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.json.Enabled(ctx, level) || h.otel.Enabled(ctx, level)
}

func (h *dualHandler) Handle(ctx context.Context, record slog.Record) error {
	var jsonErr, otelErr error

	if h.json.Enabled(ctx, record.Level) {
		jsonRecord := record.Clone()

		spanContext := trace.SpanContextFromContext(ctx)
		if spanContext.IsValid() {
			jsonRecord.AddAttrs(
				slog.String("trace_id", spanContext.TraceID().String()),
				slog.String("span_id", spanContext.SpanID().String()),
			)
		}

		jsonErr = h.json.Handle(ctx, jsonRecord)
	}

	if h.otel.Enabled(ctx, record.Level) {
		otelErr = h.otel.Handle(ctx, record.Clone())
	}

	return errors.Join(jsonErr, otelErr)
}

func (h *dualHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &dualHandler{
		json: h.json.WithAttrs(attrs),
		otel: h.otel.WithAttrs(attrs),
	}
}

func (h *dualHandler) WithGroup(name string) slog.Handler {
	return &dualHandler{
		json: h.json.WithGroup(name),
		otel: h.otel.WithGroup(name),
	}
}
