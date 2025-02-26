package tracing

const (
	AttributeObjectUID        = "object.uid"
	AttributeObjectName       = "object.name"
	AttributeObjectNamespace  = "object.namespace"
	AttributeObjectApiVersion = "object.apiVersion"
	AttributeObjectKind       = "object.kind"
	AttributeObjectGVK        = "object.gvk"
	AttributeObjectGVR        = "object.gvr"
	AttributeControllerGVK    = "controller.gvk"
	AttributeControllerGVR    = "controller.gvr"
)

var (
	AttributeHandlerName = func(name string) string {
		return "handler." + name
	}
)
