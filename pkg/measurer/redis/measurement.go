package redis

import (
	"time"

	cloudresourcesv1beta1 "github.com/kyma-project/cloud-manager/api/cloud-resources/v1beta1"

	"github.com/kyma-project/kyma-metrics-collector/pkg/measurer"
)

type Measurement struct {
	AWSRedises   cloudresourcesv1beta1.AwsRedisInstanceList
	AzureRedises cloudresourcesv1beta1.AzureRedisInstanceList
	GCPRedises   cloudresourcesv1beta1.GcpRedisInstanceList
}

func (m Measurement) UM(duration time.Duration) measurer.UMData {
	// TODO implement me
	panic("implement me")
}

func (m Measurement) EDP() measurer.EDPData {
	// TODO implement me
	panic("implement me")
}

var _ measurer.Measurement = &Measurement{}
