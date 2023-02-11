package main

import (
	"context"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/uptrace/opentelemetry-go-extra/otelzap"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	trace2 "go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"os"
)

func main() {
	ctx := context.Background()

	log.SetFormatter(&log.JSONFormatter{FieldMap: log.FieldMap{"msg": "message"}})
	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel)

	l := otelzap.New(zap.NewExample())

	exp, err := newExporter("http://dockerhost:14268/api/traces")
	if err != nil {
		log.Fatal(err)
	}

	workloadName := os.Getenv("OTEL_WORKLOAD_NAME")
	if workloadName == "" {
		workloadName = "workload"
	}

	tp := trace.NewTracerProvider(
		trace.WithBatcher(exp),
		trace.WithResource(newResource(workloadName)),
	)
	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			log.Fatal(err)
		}
	}()
	otel.SetTracerProvider(tp)

	traceparent := os.Getenv("OTEL_TRACEPARENT")
	log.Println("traceparent is " + traceparent)

	carrier := propagation.MapCarrier{"traceparent": traceparent}
	propagator := propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{})
	ctx = propagator.Extract(ctx, carrier)

	ctx, span := tp.Tracer(workloadName).Start(ctx, fmt.Sprintf("ArmadaJob.%s", workloadName))
	defer span.End()

	ctx, span = tp.Tracer("workload").Start(ctx, "SayHelloWorld")
	l.Ctx(ctx).Info("Hello World")

	span.AddEvent("World has been greeted")
	span.End()

	ctx, span = tp.Tracer("workload").Start(ctx, "FinishWorkload")
	l.Ctx(ctx).Error("some error")
	defer span.End()
	l.Ctx(ctx).Info("Workload finished!")
}

func traceLogger(ctx context.Context, l *log.Entry) *log.Entry {
	span := trace2.SpanFromContext(ctx)
	return log.WithFields(log.Fields{"trace_id": span.SpanContext().TraceID(), "span_id": span.SpanContext().SpanID()})
}

func newExporter(url string) (trace.SpanExporter, error) {
	return jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(url)))
}

// newResource returns a resource describing this application.
func newResource(service string) *resource.Resource {
	r, _ := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(service),
			semconv.ServiceVersionKey.String("v0.1.0"),
			attribute.String("environment", "demo"),
		),
	)
	return r
}
