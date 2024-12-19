package otel

import (
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/kyma-project/kyma-metrics-collector/pkg/runtime"
)

const (
	instanceIDAttr      = "instance_id"
	runtimeIDAttr       = "runtime_id"
	subAccountIDAttr    = "sub_account_id"
	globalAccountIDAttr = "global_account_id"
	shootNameAttr       = "shoot_name"
	providerAttr        = "provider"
)

func SpanAttributes(runtime *runtime.Info) trace.SpanStartEventOption {
	return trace.WithAttributes(
		attribute.String(instanceIDAttr, runtime.InstanceID),
		attribute.String(runtimeIDAttr, runtime.RuntimeID),
		attribute.String(subAccountIDAttr, runtime.SubAccountID),
		attribute.String(globalAccountIDAttr, runtime.GlobalAccountID),
		attribute.String(shootNameAttr, runtime.ShootName),
		attribute.String(providerAttr, runtime.ProviderType),
	)
}
