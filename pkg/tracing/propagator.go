package tracing

import (
	"context"
	"sync"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
)

type ObjectCarrier struct {
	mu     *sync.Mutex
	Object runtime.Object
}

func HasParent(ctx context.Context) bool {
	sp := trace.SpanContextFromContext(ctx)
	return sp.IsValid() && sp.IsRemote()
}

func Extract(ctx context.Context, obj runtime.Object) context.Context {
	return otel.GetTextMapPropagator().Extract(ctx, NewObjectCarrier(obj.DeepCopyObject()))
}

func Inject(ctx context.Context, obj runtime.Object) {
	otel.GetTextMapPropagator().Inject(ctx, NewObjectCarrier(obj))
}

func (o ObjectCarrier) isNil() bool {
	return o.Object == nil
}

func NewObjectCarrier(obj runtime.Object) ObjectCarrier {
	return ObjectCarrier{
		Object: obj,
		mu:     &sync.Mutex{},
	}
}

var _ propagation.TextMapCarrier = (*ObjectCarrier)(nil)

func (o ObjectCarrier) Get(key string) string {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.isNil() {
		return ""
	}
	accessor, err := meta.Accessor(o.Object)
	if err != nil {
		logrus.Tracef("Error getting key %s from object carrier: %s", key, err)
		return ""
	}
	logrus.Tracef("accessor : %v", accessor)
	if accessor == nil {
		return ""
	}
	existingAnn := accessor.GetAnnotations()
	logrus.Tracef("existingAnn : %v", existingAnn)
	if existingAnn == nil {
		return ""
	}
	return existingAnn[key]
}

func (o ObjectCarrier) Set(key string, value string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.isNil() {
		return
	}
	accessor, err := meta.Accessor(o.Object)
	if err != nil || accessor == nil {
		logrus.Tracef("Error setting key %s from object carrier: %s", key, err)
		return
	}
	existingAnn := accessor.GetAnnotations()
	if existingAnn == nil {
		existingAnn = make(map[string]string)
	}
	existingAnn[key] = value
	accessor.SetAnnotations(existingAnn)
}

func (o ObjectCarrier) Keys() []string {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.isNil() {
		return []string{}
	}
	accessor, err := meta.Accessor(o.Object)
	if err != nil || accessor == nil {
		logrus.Tracef("Error getting keys from object carrier: %s", err)
		return []string{}
	}
	existingAnn := accessor.GetAnnotations()
	keys := make([]string, 0, len(existingAnn))
	i := 0
	for _, k := range existingAnn {
		i++
		keys[i] = k
	}
	return keys
}
