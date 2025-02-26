package tracing_test

import (
	"context"
	"testing"
	"time"

	"github.com/rancher/lasso/pkg/tracing"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func init() {
	// TODO : testable trace exporter
	traceExporter, err := stdouttrace.New()
	if err != nil {
		panic(err)
	}
	tp := tracesdk.NewTracerProvider(
		tracesdk.WithBatcher(traceExporter, tracesdk.WithBatchTimeout(1*time.Second)),
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

	sp := trace.SpanContextFromContext(spanCtx)
	require.True(t, sp.IsValid(), "object propagator failed sanity check")
	require.False(t, sp.IsRemote(), "object propagator initial trace should be local")
	ctx := otel.GetTextMapPropagator().Extract(context.TODO(), tracing.NewObjectCarrier(p))
	_, childSpan := tr.Start(ctx, "object-propagator2")

	childSpan.SetAttributes(
		attribute.Bool("modified", true),
	)
	childSpan.End()
	childSp := trace.SpanContextFromContext(ctx)

	require.True(t, childSp.IsValid(), "object propagator child span failed sanity check")
	require.True(t, childSp.IsRemote(), "object propagator child span should be local")
	span.End()
}

func TestObjectPropagatorNilMd(t *testing.T) {
	tp := otel.GetTracerProvider()
	tr := tp.Tracer("TestObjectPropagatorNilMd")
	p := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "test",
			Namespace:   "default",
			Annotations: nil,
		},
	}
	spanCtx, span := tr.Start(context.TODO(), "object-propagator-nil-md")
	span.SetAttributes(
		attribute.String("object.uid", "test"),
		attribute.Bool("object.created", true),
	)
	otel.GetTextMapPropagator().Inject(spanCtx, tracing.NewObjectCarrier(p))

	sp := trace.SpanContextFromContext(spanCtx)
	require.True(t, sp.IsValid(), "object propagator nil failed sanity check")
	require.False(t, sp.IsRemote(), "object propagator nil initial trace should be local")

	ctx := otel.GetTextMapPropagator().Extract(context.TODO(), tracing.NewObjectCarrier(p))
	_, childSpan := tr.Start(ctx, "object-propagator2-nil-md")
	childSpan.SetAttributes(
		attribute.Bool("modified", true),
	)
	childSp := trace.SpanContextFromContext(ctx)

	require.True(t, childSp.IsValid(), "distributed child span failed sanity check")
	require.True(t, childSp.IsRemote(), "distributed child span should be remote")
	childSpan.End()
	span.End()
}

func TestObjectPropagatorNil(t *testing.T) {
	tp := otel.GetTracerProvider()
	tr := tp.Tracer("TestObjectPropagatorNil")

	spanCtx, span := tr.Start(context.TODO(), "object-propagator-nil")
	span.SetAttributes(
		attribute.String("object.uid", "test"),
		attribute.Bool("object.created", true),
	)
	otel.GetTextMapPropagator().Inject(spanCtx, tracing.NewObjectCarrier(nil))
	sp := trace.SpanContextFromContext(spanCtx)
	require.True(t, sp.IsValid(), "object propagator nil failed sanity check")
	require.False(t, sp.IsRemote(), "object propagator nil initial trace should be local")
	ctx := otel.GetTextMapPropagator().Extract(context.TODO(), tracing.NewObjectCarrier(nil))
	_, childSpan := tr.Start(ctx, "object-propagator2-nil")
	childSpan.SetAttributes(
		attribute.Bool("modified", true),
	)

	require.True(t, sp.IsValid(), "object propagator nil child span failed sanity check")
	require.False(t, sp.IsRemote(), "object propagator nil child span should not be remote")
	childSpan.End()

	span.End()
}

func TestObjectPropagatorNotInstrumented(t *testing.T) {
	tp := otel.GetTracerProvider()
	tr := tp.Tracer("TestObjectPropagatorNotInstrumented")
	p := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "test",
			Namespace:   "default",
			Annotations: nil,
		},
	}
	ctx := otel.GetTextMapPropagator().Extract(context.TODO(), tracing.NewObjectCarrier(p))
	_, childSpan := tr.Start(ctx, "object-propagator2-not-instrumented")
	childSpan.SetAttributes(
		attribute.Bool("modified", true),
	)
	childSpan.End()

	require.False(t, trace.SpanContextFromContext(ctx).IsValid(), "object propagator not instrumented failed sanity check")
}

func TestObjectPropagorNestedParent(t *testing.T) {
	tp := otel.GetTracerProvider()
	tr := tp.Tracer("TestObjectPropagatorMultiParent")
	p := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "test",
			Namespace:   "default",
			Annotations: map[string]string{},
		},
	}
	spanCtx, span := tr.Start(context.TODO(), "object-propag")
	otel.GetTextMapPropagator().Inject(spanCtx, tracing.NewObjectCarrier(p))
	sp := trace.SpanContextFromContext(spanCtx)

	require.True(t, sp.IsValid(), "object propagator multi parent failed sanity check")
	require.False(t, sp.IsRemote(), "object propagator multi parent initial trace should be local")

	ctx := otel.GetTextMapPropagator().Extract(context.TODO(), tracing.NewObjectCarrier(p))
	childCtx, childSpan := tr.Start(ctx, "object-propag")
	otel.GetTextMapPropagator().Inject(childCtx, tracing.NewObjectCarrier(p))
	childSpan.End()
	childSp := trace.SpanContextFromContext(childCtx)
	require.True(t, childSp.IsValid(), "object propagator multi parent child span failed sanity check")

	childChildCtx := otel.GetTextMapPropagator().Extract(context.TODO(), tracing.NewObjectCarrier(p))
	_, childChildSpan := tr.Start(childChildCtx, "object-propag")
	childChildSpan.End()
	childChildSp := trace.SpanContextFromContext(childChildCtx)
	require.True(t, childChildSp.IsValid(), "object propagator multi parent child child span failed sanity check")
	span.End()
}
