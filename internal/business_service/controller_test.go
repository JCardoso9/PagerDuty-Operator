package business_service

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	pagerdutyv1alpha1 "gitlab.share-now.com/platform/pagerduty-operator/api/v1alpha1"
	"gitlab.share-now.com/platform/pagerduty-operator/internal/pd_utils"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var timeout time.Duration = time.Second * 13
var interval time.Duration = time.Millisecond * 250

type TestPolicyEnv struct {
	BusServiceNamespace string
	BusService          *pagerdutyv1alpha1.BusinessService
}

func setupTest() *TestPolicyEnv {
	BusServiceNamespace := "test-" + pd_utils.RandStr(5)

	err := k8sClient.Create(ctx, &core.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: BusServiceNamespace},
	})

	Expect(err).NotTo(HaveOccurred(), "failed to create test namespace")

	BusService := &pagerdutyv1alpha1.BusinessService{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "pagerduty.platform.share-now.com/v1alpha1",
			Kind:       "BusinessService",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      Default_busService_name,
			Namespace: BusServiceNamespace,
		},
		Spec: pagerdutyv1alpha1.BusinessServiceSpec{
			Name:           Default_busService_name,
			Description:    Default_busService_description,
			PointOfContact: Default_busService_pointOfContact,
			TeamID:         Default_busService_teamID,
		},
	}

	Expect(k8sClient.Create(ctx, BusService)).Should(Succeed())

	Eventually(func() bool {
		err := k8sClient.Get(
			context.Background(),
			types.NamespacedName{Name: Default_busService_name, Namespace: BusServiceNamespace},
			&pagerdutyv1alpha1.EscalationPolicy{},
		)
		if err != nil {
			return false
		}
		return true
	}, timeout, interval).Should(BeTrue())

	return &TestPolicyEnv{
		BusServiceNamespace: BusServiceNamespace,
		BusService:          BusService,
	}

}

func cleanUp(testEnv *TestPolicyEnv) {
	err := k8sClient.Delete(ctx, &core.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: testEnv.BusServiceNamespace},
	})
	Expect(err).NotTo(HaveOccurred(), "failed to delete test namespace")
}

var _ = Describe("Business Service controller", func() {

	// Define utility constants for object names and testing timeouts/durations and intervals.
	const ()

	var testEnv *TestPolicyEnv

	Context("When creating a BusService", func() {
		BeforeEach(func() {
			testEnv = setupTest()
		})

		AfterEach(func() {
			cleanUp(testEnv)
		})

		It("Should be able to create a BusService CR with correct spec", func() {
			Expect(testEnv.BusService.Spec.Name).Should(Equal(Default_busService_name))
			Expect(testEnv.BusService.Spec.Description).Should(Equal(Default_busService_description))
			Expect(testEnv.BusService.Spec.PointOfContact).Should(Equal(Default_busService_pointOfContact))
			Expect(testEnv.BusService.Spec.TeamID).Should(Equal(Default_busService_teamID))
		})

		It("Should be able to set the status to contain the BusService ID", func() {
			Eventually(func() string {
				err := k8sClient.Get(
					context.Background(),
					types.NamespacedName{Name: Default_busService_name, Namespace: testEnv.BusServiceNamespace},
					testEnv.BusService,
				)
				if err != nil {
					return ""
				}
				return testEnv.BusService.Status.BusinessServiceID
			}, timeout, interval).Should(Equal(testEnv.BusService.Spec.Name))

			Expect(testEnv.BusService.Status.Conditions[0].Status).Should(Equal(metav1.ConditionTrue))
		})
	})

	Context("When updating a BusService", func() {
		NewDescription := "new description"
		JustBeforeEach(func() {
			testEnv = setupTest()
		})

		JustAfterEach(func() {
			cleanUp(testEnv)
		})

		It("Should be able to update a BusService CR", func() {

			Eventually(func() string {
				err := k8sClient.Get(
					context.Background(),
					types.NamespacedName{Name: Default_busService_name, Namespace: testEnv.BusServiceNamespace},
					testEnv.BusService,
				)
				if err != nil {
					return ""
				}
				return testEnv.BusService.Status.BusinessServiceID
			}, timeout, interval).Should(Equal(testEnv.BusService.Spec.Name))

			testEnv.BusService.Spec.Description = NewDescription
			Expect(k8sClient.Update(ctx, testEnv.BusService)).Should(Succeed())

			Eventually(func() string {
				err := k8sClient.Get(
					context.Background(),
					types.NamespacedName{Name: Default_busService_name, Namespace: testEnv.BusServiceNamespace},
					testEnv.BusService,
				)
				if err != nil {
					return ""
				}
				return testEnv.BusService.Spec.Description
			}, timeout, interval).Should(Equal(NewDescription))

			Expect(testEnv.BusService.Status.BusinessServiceID).Should(Equal(testEnv.BusService.Spec.Name))
			Expect(testEnv.BusService.Spec.Name).Should(Equal(Default_busService_name))
			Expect(testEnv.BusService.Spec.Description).Should(Equal(NewDescription))
			Expect(testEnv.BusService.Spec.PointOfContact).Should(Equal(Default_busService_pointOfContact))
			Expect(testEnv.BusService.Spec.TeamID).Should(Equal(Default_busService_teamID))
		})
	})
})
