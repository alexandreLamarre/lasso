package tracing_test

import (
	"context"
	"testing"
	"time"

	"github.com/rancher/lasso/pkg/tracing"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/trace"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func init() {
	// TODO : testable trace exporter
	traceExporter, err := otlptracegrpc.New(
		context.Background(),
		otlptracegrpc.WithEndpoint("0.0.0.0:4317"),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		panic(err)
	}
	tp := trace.NewTracerProvider(
		trace.WithBatcher(traceExporter, trace.WithBatchTimeout(1*time.Second)),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})
}

func TestObjectPropagator(t *testing.T) {
	tp := otel.GetTracerProvider()
	tr := tp.Tracer("TestObjectPropagator")

	p := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "test",
			Namespace:   "default",
			Annotations: map[string]string{},
		},
	}

	spanCtx, span := tr.Start(context.TODO(), "object-propagator")
	span.SetAttributes(
		attribute.String("object.uid", "test"),
		attribute.Bool("object.created", true),
	)
	otel.GetTextMapPropagator().Inject(spanCtx, tracing.NewObjectCarrier(p))

	ctx := otel.GetTextMapPropagator().Extract(context.TODO(), tracing.NewObjectCarrier(p))
	_, span2 := tr.Start(ctx, "object-propagator2")
	span2.End()

	span2.SetAttributes(
		attribute.Bool("modified", true),
	)
	span.End()

	// TODO : figure out an actual way to test this using tracetest

}
