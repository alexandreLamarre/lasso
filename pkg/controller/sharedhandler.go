package controller

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rancher/lasso/pkg/metrics"
	"github.com/rancher/lasso/pkg/tracing"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
)

var (
	ErrIgnore = errors.New("ignore handler error")
)

type handlerEntry struct {
	id        int64
	name      string
	handler   SharedControllerHandler
	parentCtx context.Context
}

type SharedHandler struct {
	// keep first because arm32 needs atomic.AddInt64 target to be mem aligned
	idCounter     int64
	controllerGVR string

	lock     sync.RWMutex
	handlers []handlerEntry
}

func (h *SharedHandler) Register(ctx context.Context, name string, handler SharedControllerHandler) {
	h.lock.Lock()
	defer h.lock.Unlock()

	id := atomic.AddInt64(&h.idCounter, 1)
	h.handlers = append(h.handlers, handlerEntry{
		id:        id,
		name:      name,
		handler:   handler,
		parentCtx: ctx,
	})

	go func() {
		<-ctx.Done()

		h.lock.Lock()
		defer h.lock.Unlock()

		for i := range h.handlers {
			if h.handlers[i].id == id {
				h.handlers = append(h.handlers[:i], h.handlers[i+1:]...)
				break
			}
		}
	}()
}

func (h *SharedHandler) staticAttrs() []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.Int64("atomic ID", h.idCounter),
		attribute.String(tracing.AttributeControllerGVR, h.controllerGVR),
	}
}

func (h *SharedHandler) OnChange(ctx context.Context, key string, obj runtime.Object) error {
	var (
		errs errorList
	)
	handlers := make([]handlerEntry, len(h.handlers))
	h.lock.RLock()
	for i, x := range h.handlers {
		handlers[i] = x
	}
	h.lock.RUnlock()

	if len(handlers) == 0 {
		return errs.ToErr()
	}

	parentSpanCtx, span := controllerTracer.Start(ctx, "SharedHandler.OnChange")
	defer span.End()

	// FIXME: this doesn't really have the effect intended
	if tracing.IsDistributedTracingEnabled() {
		parentSpanCtx = tracing.Extract(parentSpanCtx, obj)
	}

	metaA, err := meta.Accessor(obj)
	if err == nil {
		span.SetAttributes(attribute.String(tracing.AttributeObjectUID, string(metaA.GetUID())))
		span.SetAttributes(attribute.String(tracing.AttributeObjectName, metaA.GetName()))
		span.SetAttributes(attribute.String(tracing.AttributeObjectNamespace, metaA.GetNamespace()))
	}
	// FIXME: this doesn't really have the effect intended
	if tracing.IsDistributedTracingEnabled() {
		tracing.Inject(parentSpanCtx, obj)
	}

	span.SetAttributes(h.staticAttrs()...)

	for _, handler := range handlers {
		var hasError bool
		reconcileStartTS := time.Now()

		spanCtx, handlerSpan := controllerTracer.Start(parentSpanCtx, handler.name)
		newObj, err := handler.handler.OnChange(spanCtx, key, obj)
		if err != nil && !errors.Is(err, ErrIgnore) {
			errs = append(errs, &handlerError{
				HandlerName: handler.name,
				Err:         err,
			})
			hasError = true
			span.SetStatus(codes.Error, err.Error())
		} else {
			span.SetStatus(codes.Ok, "handler success")
		}
		metrics.IncTotalHandlerExecutions(h.controllerGVR, handler.name, hasError)
		reconcileTime := time.Since(reconcileStartTS)
		metrics.ReportReconcileTime(h.controllerGVR, handler.name, hasError, reconcileTime.Seconds())

		if newObj != nil && !reflect.ValueOf(newObj).IsNil() {
			meta, err := meta.Accessor(newObj)
			if err == nil && meta.GetUID() != "" {
				// avoid using an empty object
				obj = newObj
			} else if err != nil {
				// assign if we can't determine metadata
				obj = newObj
			}
		}
		handlerSpan.End()
	}

	return errs.ToErr()
}

type errorList []error

func (e errorList) Error() string {
	buf := strings.Builder{}
	for _, err := range e {
		if buf.Len() > 0 {
			buf.WriteString(", ")
		}
		buf.WriteString(err.Error())
	}
	return buf.String()
}

func (e errorList) ToErr() error {
	switch len(e) {
	case 0:
		return nil
	case 1:
		return e[0]
	default:
		return e
	}
}

func (e errorList) Cause() error {
	if len(e) > 0 {
		return e[0]
	}
	return nil
}

type handlerError struct {
	HandlerName string
	Err         error
}

func (h handlerError) Error() string {
	return fmt.Sprintf("handler %s: %v", h.HandlerName, h.Err)
}

func (h handlerError) Cause() error {
	return h.Err
}
