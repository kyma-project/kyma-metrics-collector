package commons

import (
	cloudresourcesv1beta1 "github.com/kyma-project/cloud-manager/api/cloud-resources/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func SetupScheme() (*runtime.Scheme, error) {
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		return nil, err
	}

	if err := cloudresourcesv1beta1.AddToScheme(scheme); err != nil {
		return nil, err
	}

	return scheme, nil
}
