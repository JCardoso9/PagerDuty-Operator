package escalation_policy

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	pagerdutyv1alpha1 "gitlab.share-now.com/platform/pagerduty-operator/api/v1alpha1"
	"gitlab.share-now.com/platform/pagerduty-operator/internal/pd_utils"
	"gitlab.share-now.com/platform/pagerduty-operator/internal/typeinfo"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var timeout time.Duration = time.Second * 13
var interval time.Duration = time.Millisecond * 250

type TestPolicyEnv struct {
	PolicyNamespace string
	Policy          *pagerdutyv1alpha1.EscalationPolicy
}

func setupTest() *TestPolicyEnv {
	policyNamespace := "test-" + pd_utils.RandStr(5)

	err := k8sClient.Create(ctx, &core.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: policyNamespace},
	})

	Expect(err).NotTo(HaveOccurred(), "failed to create test namespace")

	policy := &pagerdutyv1alpha1.EscalationPolicy{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "pagerduty.platform.share-now.com/v1alpha1",
			Kind:       "EscalationPolicy",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      Default_policy_name,
			Namespace: policyNamespace,
		},
		Spec: pagerdutyv1alpha1.EscalationPolicySpec{
			Name:                       Default_policy_name,
			Description:                Default_policy_description,
			NumLoops:                   Default_num_loops,
			OnCallHandoffNotifications: Default_on_call_handoff_notifications,
			EscalationRules: []typeinfo.K8sEscalationRule{
				{
					Targets: typeinfo.UserIDList{
						typeinfo.UserID("MOCKUSERID"),
					},
					Delay: 5,
				},
			},
		},
	}

	Expect(k8sClient.Create(ctx, policy)).Should(Succeed())

	Eventually(func() bool {
		err := k8sClient.Get(
			context.Background(),
			types.NamespacedName{Name: Default_policy_name, Namespace: policyNamespace},
			&pagerdutyv1alpha1.EscalationPolicy{},
		)
		if err != nil {
			return false
		}
		return true
	}, timeout, interval).Should(BeTrue())

	return &TestPolicyEnv{
		PolicyNamespace: policyNamespace,
		Policy:          policy,
	}

}

func cleanUp(testEnv *TestPolicyEnv) {
	err := k8sClient.Delete(ctx, &core.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: testEnv.PolicyNamespace},
	})
	Expect(err).NotTo(HaveOccurred(), "failed to delete test namespace")
}

var _ = Describe("EscalationPolicy controller", func() {

	// Define utility constants for object names and testing timeouts/durations and intervals.
	const ()

	var testEnv *TestPolicyEnv

	Context("When creating a Policy", func() {
		BeforeEach(func() {
			testEnv = setupTest()
		})

		AfterEach(func() {
			cleanUp(testEnv)
		})

		It("Should be able to create a policy CR with correct spec", func() {
			Expect(testEnv.Policy.Spec.Description).Should(Equal(Default_policy_description))
			Expect(testEnv.Policy.Spec.Name).Should(Equal(Default_policy_name))
			Expect(testEnv.Policy.Spec.NumLoops).Should(Equal(Default_num_loops))
			Expect(testEnv.Policy.Spec.OnCallHandoffNotifications).Should(Equal(Default_on_call_handoff_notifications))
			Expect(testEnv.Policy.Spec.EscalationRules).Should(Equal(typeinfo.K8sEscalationRuleList{
				{
					Targets: typeinfo.UserIDList{
						typeinfo.UserID("MOCKUSERID"),
					},
					Delay: 5,
				},
			}))
		})

		It("Should be able to set the status to contain the policy ID", func() {
			Eventually(func() string {
				err := k8sClient.Get(
					context.Background(),
					types.NamespacedName{Name: Default_policy_name, Namespace: testEnv.PolicyNamespace},
					testEnv.Policy,
				)
				if err != nil {
					return ""
				}
				return testEnv.Policy.Status.PolicyID
			}, timeout, interval).Should(Equal(testEnv.Policy.Spec.Name))

			Expect(testEnv.Policy.Status.Conditions[0].Status).Should(Equal(metav1.ConditionTrue))
		})
	})

	Context("When updating a Policy", func() {
		NewDescription := "new description"
		JustBeforeEach(func() {
			testEnv = setupTest()
		})

		JustAfterEach(func() {
			cleanUp(testEnv)
		})

		It("Should be able to update a policy CR", func() {

			Eventually(func() string {
				err := k8sClient.Get(
					context.Background(),
					types.NamespacedName{Name: Default_policy_name, Namespace: testEnv.PolicyNamespace},
					testEnv.Policy,
				)
				if err != nil {
					return ""
				}
				return testEnv.Policy.Status.PolicyID
			}, timeout, interval).Should(Equal(testEnv.Policy.Spec.Name))

			testEnv.Policy.Spec.Description = NewDescription
			Expect(k8sClient.Update(ctx, testEnv.Policy)).Should(Succeed())

			Eventually(func() string {
				err := k8sClient.Get(
					context.Background(),
					types.NamespacedName{Name: Default_policy_name, Namespace: testEnv.PolicyNamespace},
					testEnv.Policy,
				)
				if err != nil {
					return ""
				}
				return testEnv.Policy.Spec.Description
			}, timeout, interval).Should(Equal(NewDescription))

			Expect(testEnv.Policy.Status.PolicyID).Should(Equal(testEnv.Policy.Spec.Name))
			Expect(testEnv.Policy.Spec.Name).Should(Equal(Default_policy_name))
			Expect(testEnv.Policy.Spec.NumLoops).Should(Equal(Default_num_loops))
			Expect(testEnv.Policy.Spec.OnCallHandoffNotifications).Should(Equal(Default_on_call_handoff_notifications))
			Expect(testEnv.Policy.Spec.EscalationRules).Should(Equal(typeinfo.K8sEscalationRuleList{
				{
					Targets: typeinfo.UserIDList{
						typeinfo.UserID("MOCKUSERID"),
					},
					Delay: 5,
				},
			}))
		})

	})
})
