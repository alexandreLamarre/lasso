package tracing

import (
	"context"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
)

type ObjectCarrier struct {
	Object runtime.Object
}

func Extract(ctx context.Context, obj runtime.Object) context.Context {
	return otel.GetTextMapPropagator().Extract(ctx, NewObjectCarrier(obj))
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
	}
}

var _ propagation.TextMapCarrier = (*ObjectCarrier)(nil)

func (o ObjectCarrier) Get(key string) string {
	if o.isNil() {
		return ""
	}
	accessor, err := meta.Accessor(o.Object)
	if err != nil {
		logrus.Tracef("Error getting key %s from object carrier: %s", key, err)
		return ""
	}
	existingAnn := accessor.GetAnnotations()
	return existingAnn[key]
}

func (o ObjectCarrier) Set(key string, value string) {
	if o.isNil() {
		return
	}
	accessor, err := meta.Accessor(o.Object)
	if err != nil {
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
	if o.isNil() {
		return nil
	}
	accessor, err := meta.Accessor(o.Object)
	if err != nil {
		logrus.Tracef("Error getting keys from object carrier: %s", err)
		return nil
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
