package trace

import (
	"context"
	"time"

	"github.com/evanofslack/analogdb/config"
	"github.com/evanofslack/analogdb/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	connectTimeout  = 5 * time.Second
	shutdownTimeout = 5 * time.Second
)

type Tracer struct {
	logger         *logger.Logger
	config         *config.Config
	tracerProvider *sdktrace.TracerProvider
}

func New(logger *logger.Logger, config *config.Config) (*Tracer, error) {

	logger.Debug().Msg("Creating new otel tracer")

	tracer := &Tracer{
		logger: logger,
		config: config,
	}

	// Always propagate trace context
	propagator := propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{}, propagation.Baggage{})
	otel.SetTextMapPropagator(propagator)

	logger.Info().Msg("Initalized otel tracer")

	return tracer, nil
}

func (tracer *Tracer) StartExporter() error {

	tracer.logger.Debug().Msg("Starting up tracing exporter")

	endpoint := tracer.config.Tracing.Endpoint
	if endpoint == "" {
		tracer.logger.Warn().Msg("Tracing endpoint is not set, skipping otel exporter startup")
		return nil

	}

	ctx, cancel := context.WithTimeout(context.Background(), connectTimeout)
	defer cancel()

	service := tracer.config.App.Name
	version := tracer.config.App.Version
	tracer.logger.Debug().Str("service-name", service).Str("version", version).Msg("Creating new tracing resource")

	resource, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(service),
			attribute.String("version", version),
		),
	)

	if err != nil {
		tracer.logger.Err(err).Msg("Failed to created tracing resource")
		return err
	}

	endpoint := tracer.config.Tracing.Endpoint
	tracer.logger.Debug().Str("endpoint", endpoint).Msg("Dialing GRPC endpoint for OTLP exporter")

	conn, err := grpc.DialContext(ctx, endpoint, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		tracer.logger.Err(err).Str("endpoint", endpoint).Msg("Failed to dial GRPC endpoint for OTLP exporter")
	}

	tracer.logger.Debug().Msg("Creating new OTLP exporter")
	exporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
	if err != nil {
		tracer.logger.Err(err).Msg("Failed to create new OTLP exporter")
	}

	tracer.logger.Debug().Msg("Creating new tracing provider")
	batchSpanProcessor := sdktrace.NewBatchSpanProcessor(exporter)
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(resource),
		sdktrace.WithSpanProcessor(batchSpanProcessor),
	)

	tracer.tracerProvider = tracerProvider
	otel.SetTracerProvider(tracerProvider)

	tracer.logger.Info().Str("endpoint", endpoint).Msg("Started tracing exporter")

	return nil

}

func (tracer *Tracer) Close() error {
	tracer.logger.Debug().Msg("Starting tracing exporter close")
	defer tracer.logger.Info().Msg("Closed tracing exporter")

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	return tracer.tracerProvider.Shutdown(ctx)
}
