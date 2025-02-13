package pvc

import (
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apiresource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kyma-project/kyma-metrics-collector/pkg/resource"
)

func TestScan_EDP(t *testing.T) {
	tests := []struct {
		name     string
		pvcs     corev1.PersistentVolumeClaimList
		expected resource.EDPMeasurement
	}{
		{
			name: "no pvcs",
			pvcs: corev1.PersistentVolumeClaimList{},
			expected: resource.EDPMeasurement{
				ProvisionedVolumes: resource.ProvisionedVolumes{
					SizeGbTotal:   0,
					SizeGbRounded: 0,
					Count:         0,
				},
			},
		},
		{
			name: "single non-nfs pvc",
			pvcs: corev1.PersistentVolumeClaimList{
				Items: []corev1.PersistentVolumeClaim{
					{
						Status: corev1.PersistentVolumeClaimStatus{
							Phase:    corev1.ClaimBound,
							Capacity: corev1.ResourceList{corev1.ResourceStorage: apiresource.MustParse("10Gi")},
						},
					},
				},
			},
			expected: resource.EDPMeasurement{
				ProvisionedVolumes: resource.ProvisionedVolumes{
					SizeGbTotal:   10,
					SizeGbRounded: 32, // Rounded to 32 GiB
					Count:         1,
				},
			},
		},
		{
			name: "single nfs pvc",
			pvcs: corev1.PersistentVolumeClaimList{
				Items: []corev1.PersistentVolumeClaim{
					{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"app.kubernetes.io/component":  "cloud-manager",
								"app.kubernetes.io/part-of":    "kyma",
								"app.kubernetes.io/managed-by": "cloud-manager",
							}, // NFS labels
						},
						Status: corev1.PersistentVolumeClaimStatus{
							Phase:    corev1.ClaimBound,
							Capacity: corev1.ResourceList{corev1.ResourceStorage: apiresource.MustParse("20Gi")},
						},
					},
				},
			},
			expected: resource.EDPMeasurement{
				ProvisionedVolumes: resource.ProvisionedVolumes{
					SizeGbTotal:   60, // Multiplied by nfsPriceMultiplier (3)
					SizeGbRounded: 64, // Rounded to 64 GiB
					Count:         1,
				},
			},
		},
		{
			name: "mixed pvcs",
			pvcs: corev1.PersistentVolumeClaimList{
				Items: []corev1.PersistentVolumeClaim{
					{
						Status: corev1.PersistentVolumeClaimStatus{
							Phase:    corev1.ClaimBound,
							Capacity: corev1.ResourceList{corev1.ResourceStorage: apiresource.MustParse("10Gi")},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"app.kubernetes.io/component":  "cloud-manager",
								"app.kubernetes.io/part-of":    "kyma",
								"app.kubernetes.io/managed-by": "cloud-manager",
							}, // NFS labels
						},
						Status: corev1.PersistentVolumeClaimStatus{
							Phase:    corev1.ClaimBound,
							Capacity: corev1.ResourceList{corev1.ResourceStorage: apiresource.MustParse("20Gi")},
						},
					},
				},
			},
			expected: resource.EDPMeasurement{
				ProvisionedVolumes: resource.ProvisionedVolumes{
					SizeGbTotal:   70, // 10 + (20*3)
					SizeGbRounded: 96, // 32 (rounded 10) + 64 (rounded 60)
					Count:         2,
				},
			},
		},
		{
			name: "cloud-manager with no nfs capacity label",
			pvcs: corev1.PersistentVolumeClaimList{
				Items: []corev1.PersistentVolumeClaim{
					{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"app.kubernetes.io/component":  "cloud-manager",
								"app.kubernetes.io/part-of":    "kyma",
								"app.kubernetes.io/managed-by": "cloud-manager",
							}, // NFS labels
						},
						Status: corev1.PersistentVolumeClaimStatus{
							Phase:    corev1.ClaimBound,
							Capacity: corev1.ResourceList{corev1.ResourceStorage: apiresource.MustParse("20Gi")},
						},
					},
				},
			},
			expected: resource.EDPMeasurement{
				ProvisionedVolumes: resource.ProvisionedVolumes{
					SizeGbTotal:   60, // 3 * 20
					SizeGbRounded: 64, // 2 * 32
					Count:         1,
				},
			},
		},
		{
			name: "cloud-manager with nfs capacity label taking priority",
			pvcs: corev1.PersistentVolumeClaimList{
				Items: []corev1.PersistentVolumeClaim{
					{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"app.kubernetes.io/component":                              "cloud-manager",
								"app.kubernetes.io/part-of":                                "kyma",
								"app.kubernetes.io/managed-by":                             "cloud-manager",
								"cloud-resources.kyma-project.io/nfsVolumeStorageCapacity": "40Gi",
							}, // NFS labels
						},
						Status: corev1.PersistentVolumeClaimStatus{
							Phase:    corev1.ClaimBound,
							Capacity: corev1.ResourceList{corev1.ResourceStorage: apiresource.MustParse("20Gi")},
						},
					},
				},
			},
			expected: resource.EDPMeasurement{
				ProvisionedVolumes: resource.ProvisionedVolumes{
					SizeGbTotal:   120, // 3 * 40
					SizeGbRounded: 128, // 32 * 4
					Count:         1,
				},
			},
		},
		{
			name: "cloud-manager with unparasable nfs capacity label using pvc capacity as fallback",
			pvcs: corev1.PersistentVolumeClaimList{
				Items: []corev1.PersistentVolumeClaim{
					{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"app.kubernetes.io/component":                              "cloud-manager",
								"app.kubernetes.io/part-of":                                "kyma",
								"app.kubernetes.io/managed-by":                             "cloud-manager",
								"cloud-resources.kyma-project.io/nfsVolumeStorageCapacity": "invalid value",
							}, // NFS labels
						},
						Status: corev1.PersistentVolumeClaimStatus{
							Phase:    corev1.ClaimBound,
							Capacity: corev1.ResourceList{corev1.ResourceStorage: apiresource.MustParse("20Gi")},
						},
					},
				},
			},
			expected: resource.EDPMeasurement{
				ProvisionedVolumes: resource.ProvisionedVolumes{
					SizeGbTotal:   60, // 3 * 20
					SizeGbRounded: 64, // 2 * 32
					Count:         1,
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			scan := &Scan{
				pvcs: test.pvcs,
			}
			actual, err := scan.EDP()
			require.NoError(t, err)
			require.Equal(t, test.expected, actual)
		})
	}
}
