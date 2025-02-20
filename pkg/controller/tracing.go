package controller

import "go.opentelemetry.io/otel"

var (
	controllerTracer = otel.Tracer("SharedController")
)
