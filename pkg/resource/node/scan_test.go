package node

import (
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kyma-project/kyma-metrics-collector/pkg/config"
	"github.com/kyma-project/kyma-metrics-collector/pkg/resource"
	kmctesting "github.com/kyma-project/kyma-metrics-collector/pkg/testing"
)

func TestScan_EDP(t *testing.T) {
	specs := &config.PublicCloudSpecs{
		Providers: config.Providers{
			AWS: map[string]config.Feature{
				"t2.micro":            {CpuCores: 1, Memory: 1},
				"m5.large":            {CpuCores: 2, Memory: 8},
				"gpu.fake.fractional": {CpuCores: 1234.1234455, Memory: 1234.1234455},
			},
			Azure: map[string]config.Feature{
				"a1.standard": {CpuCores: 1, Memory: 1.75},
			},
			GCP: map[string]config.Feature{
				"n1-standard-1": {CpuCores: 1, Memory: 3.75},
			},
		},
	}

	tests := []struct {
		name          string
		provider      string
		list          metav1.PartialObjectMetadataList
		expectedEDP   resource.EDPMeasurement
		expectedError error
	}{
		{
			name:     "no list",
			provider: config.AWS,
			list:     metav1.PartialObjectMetadataList{},
			expectedEDP: resource.EDPMeasurement{
				ProvisionedCPUs:  0,
				ProvisionedRAMGb: 0,
				VMTypes:          nil,
			},
			expectedError: nil,
		},
		{
			name:     "single valid aws node",
			provider: config.AWS,
			list: metav1.PartialObjectMetadataList{
				Items: []metav1.PartialObjectMetadata{
					{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{"node.kubernetes.io/instance-type": "t2.micro"},
						},
					},
				},
			},
			expectedEDP: resource.EDPMeasurement{
				ProvisionedCPUs:  1,
				ProvisionedRAMGb: 1,
				VMTypes: []resource.VMType{
					{Name: "t2.micro", Count: 1},
				},
			},
			expectedError: nil,
		},
		{
			name:     "single 'GPU' node",
			provider: config.AWS,
			list: metav1.PartialObjectMetadataList{
				Items: []metav1.PartialObjectMetadata{
					{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{"node.kubernetes.io/instance-type": "gpu.fake.fractional"},
						},
					},
				},
			},
			expectedEDP: resource.EDPMeasurement{
				ProvisionedCPUs:  1234.1234455,
				ProvisionedRAMGb: 1234.1234455,
				VMTypes: []resource.VMType{
					{Name: "gpu.fake.fractional", Count: 1},
				},
			},
			expectedError: nil,
		},
		{
			name:     "multiple 'GPU' nodes",
			provider: config.AWS,
			list: metav1.PartialObjectMetadataList{
				Items: []metav1.PartialObjectMetadata{
					{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{"node.kubernetes.io/instance-type": "gpu.fake.fractional"},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{"node.kubernetes.io/instance-type": "gpu.fake.fractional"},
						},
					},
				},
			},
			expectedEDP: resource.EDPMeasurement{
				ProvisionedCPUs:  2468.246891,
				ProvisionedRAMGb: 2468.246891,
				VMTypes: []resource.VMType{
					{Name: "gpu.fake.fractional", Count: 2},
				},
			},
			expectedError: nil,
		},
		{
			name:     "unknown node type",
			provider: config.AWS,
			list: metav1.PartialObjectMetadataList{
				Items: []metav1.PartialObjectMetadata{
					{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{"node.kubernetes.io/instance-type": "unknown-type"},
						},
					},
				},
			},
			expectedEDP: resource.EDPMeasurement{
				ProvisionedCPUs:  0,
				ProvisionedRAMGb: 0,
				VMTypes:          nil,
			},
			expectedError: ErrUnknownVM,
		},
		{
			name:     "mixed valid and unknown list",
			provider: config.AWS,
			list: metav1.PartialObjectMetadataList{
				Items: []metav1.PartialObjectMetadata{
					{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{"node.kubernetes.io/instance-type": "t2.micro"},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{"node.kubernetes.io/instance-type": "unknown-type"},
						},
					},
				},
			},
			expectedEDP: resource.EDPMeasurement{
				ProvisionedCPUs:  1,
				ProvisionedRAMGb: 1,
				VMTypes: []resource.VMType{
					{Name: "t2.micro", Count: 1},
				},
			},
			expectedError: ErrUnknownVM,
		},
		{
			name:     "multiple valid list",
			provider: config.AWS,
			list: metav1.PartialObjectMetadataList{
				Items: []metav1.PartialObjectMetadata{
					{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{"node.kubernetes.io/instance-type": "t2.micro"},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{"node.kubernetes.io/instance-type": "m5.large"},
						},
					},
				},
			},
			expectedEDP: resource.EDPMeasurement{
				ProvisionedCPUs:  3,
				ProvisionedRAMGb: 9,
				VMTypes: []resource.VMType{
					{Name: "t2.micro", Count: 1},
					{Name: "m5.large", Count: 1},
				},
			},
			expectedError: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			scan := &Scan{
				providerType: test.provider,
				specs:        specs,
				list:         test.list,
			}

			actualEDP, err := scan.EDP()

			require.InDelta(t, test.expectedEDP.ProvisionedCPUs, actualEDP.ProvisionedCPUs, kmctesting.Delta)
			require.InDelta(t, test.expectedEDP.ProvisionedRAMGb, actualEDP.ProvisionedRAMGb, kmctesting.Delta)
			require.ElementsMatch(t, test.expectedEDP.VMTypes, actualEDP.VMTypes)

			if test.expectedError != nil {
				require.ErrorIs(t, err, test.expectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
