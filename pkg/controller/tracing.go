package controller

import (
	"context"

	"github.com/rancher/lasso/pkg/tracing"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"k8s.io/apimachinery/pkg/runtime"
)

var (
	controllerTracer = otel.GetTracerProvider().Tracer("SharedController")
)

func (h *SharedHandler) setupTracingCtx(ctx context.Context, obj runtime.Object) context.Context {
	return tracing.Extract(ctx, obj)
}

func (h *SharedHandler) setupSpan(ctx context.Context) (context.Context, trace.Span) {
	if tracing.HasParent(ctx) {
		return controllerTracer.Start(
			ctx,
			"sharedhandler.OnChange",
			trace.WithLinks(trace.Link{
				SpanContext: trace.SpanContextFromContext(ctx),
			}),
		)
	}
	return controllerTracer.Start(ctx, "sharedhandler.OnChange")
}
