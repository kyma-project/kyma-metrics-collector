package redis

import (
	"context"
	"strconv"
	"testing"

	cloudresourcesv1beta1 "github.com/kyma-project/cloud-manager/api/cloud-resources/v1beta1"
	"github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus/testutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"

	kmccache "github.com/kyma-project/kyma-metrics-collector/pkg/cache"
	skrcommons "github.com/kyma-project/kyma-metrics-collector/pkg/skr/commons"
	kmctesting "github.com/kyma-project/kyma-metrics-collector/pkg/testing"
)

func TestList(t *testing.T) {
	g := gomega.NewGomegaWithT(t)
	// given
	ctx := context.Background()
	givenShootInfo := kmccache.Record{
		InstanceID:      "adccb200-6052-4192-8adf-785b8a5af306",
		RuntimeID:       "fe5ab5d6-5b0b-4b70-9644-7f89d230b516",
		SubAccountID:    "1ae0dbe1-d13d-4e39-bed4-7c83364084d5",
		GlobalAccountID: "0c22f798-e572-4fc7-a502-cd825c742ff6",
		ShootName:       "c-987654",
	}

	awsRedises := kmctesting.AWSRedisList()
	azureRedises := kmctesting.AzureRedisList()
	gcpRedises := kmctesting.GCPRedisList()

	client := newClientWithFakes(
		t,
		awsRedises,
		azureRedises,
		gcpRedises,
		givenShootInfo,
	)

	// when
	gotRedisList, err := client.List(ctx)

	// then
	g.Expect(err).Should(gomega.BeNil())
	g.Expect(gotRedisList.AWS.Items).To(gomega.Equal(awsRedises.Items))
	g.Expect(gotRedisList.Azure.Items).To(gomega.Equal(azureRedises.Items))
	g.Expect(gotRedisList.GCP.Items).To(gomega.Equal(gcpRedises.Items))

	// ensure metrics.
	gotMetrics, err := skrcommons.TotalQueriesMetric.GetMetricWithLabelValues(
		skrcommons.ListingRedisesAWSAction,
		strconv.FormatBool(true),
		givenShootInfo.ShootName,
		givenShootInfo.InstanceID,
		givenShootInfo.RuntimeID,
		givenShootInfo.SubAccountID,
		givenShootInfo.GlobalAccountID,
	)
	g.Expect(err).Should(gomega.BeNil())
	g.Expect(testutil.ToFloat64(gotMetrics)).Should(gomega.Equal(float64(1)))

	gotMetrics, err = skrcommons.TotalQueriesMetric.GetMetricWithLabelValues(
		skrcommons.ListingRedisesAzureAction,
		strconv.FormatBool(true),
		givenShootInfo.ShootName,
		givenShootInfo.InstanceID,
		givenShootInfo.RuntimeID,
		givenShootInfo.SubAccountID,
		givenShootInfo.GlobalAccountID,
	)
	g.Expect(err).Should(gomega.BeNil())
	g.Expect(testutil.ToFloat64(gotMetrics)).Should(gomega.Equal(float64(1)))

	actionLabels := []string{skrcommons.ListingRedisesAWSAction, skrcommons.ListingRedisesAzureAction, skrcommons.ListingRedisesGCPAction}
	for _, actionLabel := range actionLabels {
		gotMetrics, err = skrcommons.TotalQueriesMetric.GetMetricWithLabelValues(
			actionLabel,
			strconv.FormatBool(true),
			givenShootInfo.ShootName,
			givenShootInfo.InstanceID,
			givenShootInfo.RuntimeID,
			givenShootInfo.SubAccountID,
			givenShootInfo.GlobalAccountID,
		)

		g.Expect(err).Should(gomega.BeNil())
		g.Expect(testutil.ToFloat64(gotMetrics)).Should(gomega.Equal(float64(1)))
	}

	// given - another case.
	// Delete all the AWS redises
	for _, svc := range awsRedises.Items {
		err := client.AWSRedisClient.Namespace(svc.Namespace).Delete(ctx, svc.Name, metav1.DeleteOptions{})
		g.Expect(err).Should(gomega.BeNil())
	}

	// when
	gotRedisList, err = client.List(ctx)

	// then
	g.Expect(err).Should(gomega.BeNil())
	g.Expect(len(gotRedisList.AWS.Items)).To(gomega.Equal(0))
	// Azure and GCP redises should be the same.
	g.Expect(gotRedisList.Azure.Items).To(gomega.Equal(azureRedises.Items))
	g.Expect(gotRedisList.GCP.Items).To(gomega.Equal(gcpRedises.Items))

	// ensure metrics.
	for _, actionLabel := range actionLabels {
		gotMetrics, err = skrcommons.TotalQueriesMetric.GetMetricWithLabelValues(
			actionLabel,
			strconv.FormatBool(true),
			givenShootInfo.ShootName,
			givenShootInfo.InstanceID,
			givenShootInfo.RuntimeID,
			givenShootInfo.SubAccountID,
			givenShootInfo.GlobalAccountID,
		)

		g.Expect(err).Should(gomega.BeNil())
		g.Expect(testutil.ToFloat64(gotMetrics)).Should(gomega.Equal(float64(2)))
	}
}

func newClientWithFakes(
	t *testing.T,
	awsRedises *cloudresourcesv1beta1.AwsRedisInstanceList,
	azureRedises *cloudresourcesv1beta1.AzureRedisInstanceList,
	gcpRedises *cloudresourcesv1beta1.GcpRedisInstanceList,
	shootInfo kmccache.Record,
) *Client {
	t.Helper()

	scheme, err := skrcommons.SetupScheme()
	if err != nil {
		t.Errorf("failed to setup scheme: %v", err)
	}

	dynamicClient := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(scheme,
		map[schema.GroupVersionResource]string{
			AWSRedisGVR:   "AwsRedisInstanceList",
			AzureRedisGVR: "AzureRedisInstanceList",
			GCPRedisGVR:   "GcpRedisInstanceList",
		}, awsRedises, azureRedises, gcpRedises)

	return &Client{
		AWSRedisClient:   dynamicClient.Resource(AWSRedisGVR),
		AzureRedisClient: dynamicClient.Resource(AzureRedisGVR),
		GCPRedisClient:   dynamicClient.Resource(GCPRedisGVR),
		ShootInfo:        shootInfo,
	}
}
