package vsc

import (
	"testing"

	v1 "github.com/kubernetes-csi/external-snapshotter/client/v6/apis/volumesnapshot/v1"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"github.com/kyma-project/kyma-metrics-collector/pkg/resource"
)

func TestScan_EDP(t *testing.T) {
	tests := []struct {
		name          string
		vscs          v1.VolumeSnapshotContentList
		expected      resource.EDPMeasurement
		expextedError error
	}{
		{
			name: "no vscs",
			vscs: v1.VolumeSnapshotContentList{},
			expected: resource.EDPMeasurement{
				ProvisionedVolumes: resource.ProvisionedVolumes{
					SizeGbTotal:   0,
					SizeGbRounded: 0,
					Count:         0,
				},
			},
		},
		{
			name: "single vsc",
			vscs: v1.VolumeSnapshotContentList{
				Items: []v1.VolumeSnapshotContent{
					{
						Status: &v1.VolumeSnapshotContentStatus{
							ReadyToUse:  ptr.To(true),
							RestoreSize: ptr.To(int64(10737418240)), // 10GB
						},
					},
				},
			},
			expected: resource.EDPMeasurement{
				ProvisionedVolumes: resource.ProvisionedVolumes{
					SizeGbTotal:   10,
					SizeGbRounded: 32, // Rounded to 32 GB
					Count:         1,
				},
			},
		},
		{
			name: "mixed vscs",
			vscs: v1.VolumeSnapshotContentList{
				Items: []v1.VolumeSnapshotContent{
					{
						Status: &v1.VolumeSnapshotContentStatus{
							ReadyToUse:  ptr.To(true),
							RestoreSize: ptr.To(int64(10737418240)), // 10GB
						},
					},
					{
						Status: &v1.VolumeSnapshotContentStatus{
							ReadyToUse:  ptr.To(true),
							RestoreSize: ptr.To(int64(2 * 10737418240)), // 20GB
						},
					},
				},
			},
			expected: resource.EDPMeasurement{
				ProvisionedVolumes: resource.ProvisionedVolumes{
					SizeGbTotal:   30, // 10 + 20
					SizeGbRounded: 64, // 32 (rounded 10) + 32 (rounded 20)
					Count:         2,
				},
			},
		},
		{
			name: "multiple vscs with different sizes",
			vscs: v1.VolumeSnapshotContentList{
				Items: []v1.VolumeSnapshotContent{
					{
						Status: &v1.VolumeSnapshotContentStatus{
							ReadyToUse:  ptr.To(true),
							RestoreSize: ptr.To(int64(5 * 1073741824)), // 5GB
						},
					},
					{
						Status: &v1.VolumeSnapshotContentStatus{
							ReadyToUse:  ptr.To(true),
							RestoreSize: ptr.To(int64(15 * 1073741824)), // 15GB
						},
					},
				},
			},
			expected: resource.EDPMeasurement{
				ProvisionedVolumes: resource.ProvisionedVolumes{
					SizeGbTotal:   20, // 5 + 15
					SizeGbRounded: 64, // 32 (rounded 5) + 32 (rounded 15)
					Count:         2,
				},
			},
		},
		{
			name: "vscs with not ready status",
			vscs: v1.VolumeSnapshotContentList{
				Items: []v1.VolumeSnapshotContent{
					{
						Status: &v1.VolumeSnapshotContentStatus{
							ReadyToUse:  ptr.To(false),
							RestoreSize: ptr.To(int64(10737418240)), // 10GB
						},
					},
				},
			},
			expected: resource.EDPMeasurement{
				ProvisionedVolumes: resource.ProvisionedVolumes{
					SizeGbTotal:   0,
					SizeGbRounded: 0,
					Count:         0,
				},
			},
		},
		{
			name: "vscs with no status",
			vscs: v1.VolumeSnapshotContentList{
				Items: []v1.VolumeSnapshotContent{
					{ObjectMeta: metav1.ObjectMeta{Name: "vsc1"}},
				},
			},
			expected: resource.EDPMeasurement{
				ProvisionedVolumes: resource.ProvisionedVolumes{
					SizeGbTotal:   0,
					SizeGbRounded: 0,
					Count:         0,
				},
			},
			expextedError: nil,
		},
		{
			name: "vscs no restore size",
			vscs: v1.VolumeSnapshotContentList{
				Items: []v1.VolumeSnapshotContent{
					{
						ObjectMeta: metav1.ObjectMeta{Name: "vsc1"},
						Status: &v1.VolumeSnapshotContentStatus{
							ReadyToUse:  ptr.To(true),
							RestoreSize: nil,
						},
					},
				},
			},
			expected: resource.EDPMeasurement{
				ProvisionedVolumes: resource.ProvisionedVolumes{
					SizeGbTotal:   0,
					SizeGbRounded: 0,
					Count:         0,
				},
			},
			expextedError: ErrRestoreSizeNotSet,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			scan := &Scan{
				vscs: test.vscs,
			}

			actual, err := scan.EDP()
			if test.expextedError != nil {
				require.ErrorIs(t, err, test.expextedError)
			}

			require.Equal(t, test.expected, actual)
		})
	}
}
