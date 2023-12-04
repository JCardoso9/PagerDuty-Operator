package escalation_policy

import (
	"context"

	"github.com/PagerDuty/go-pagerduty"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gitlab.share-now.com/platform/pagerduty-operator/api/v1alpha1"
	"gitlab.share-now.com/platform/pagerduty-operator/internal/typeinfo"
)

func compareLocalToUpstream(pdPolicy *pagerduty.EscalationPolicy, k8sPolicy *v1alpha1.EscalationPolicy) {
	GinkgoHelper()
	Expect(pdPolicy.Name).To(Equal(k8sPolicy.Spec.Name))
	Expect(pdPolicy.Description).To(Equal(k8sPolicy.Spec.Description))
	Expect(pdPolicy.NumLoops).To(Equal(k8sPolicy.Spec.NumLoops))
	Expect(pdPolicy.OnCallHandoffNotifications).To(Equal(k8sPolicy.Spec.OnCallHandoffNotifications))
	assertEscalationRulesMatch(pdPolicy.EscalationRules, k8sPolicy.Spec.EscalationRules)
}

func assertEscalationRulesMatch(pdEscalationRules []pagerduty.EscalationRule, k8sEscalationRules typeinfo.K8sEscalationRuleList) {
	GinkgoHelper()
	Expect(len(pdEscalationRules)).To(Equal(len(k8sEscalationRules)))
	for i, rule := range k8sEscalationRules {
		Expect(rule.Delay).To(Equal(pdEscalationRules[i].Delay))
		Expect(len(rule.Targets)).To(Equal(len(pdEscalationRules[i].Targets)))
		for j, target := range rule.Targets {
			Expect(string(target)).To(Equal(pdEscalationRules[i].Targets[j].ID))
		}
	}
}

var _ = Describe("Escalation policy adapter tests", func() {

	const (
		NumLoops                   uint   = 2
		PolicyName                 string = "test-policy"
		PolicyDescription          string = "test-policy-description"
		OnCallHandoffNotifications string = "if_has_services"
		MockUserID                 string = "MOCKUSERID"
		Delay                      uint   = 5
	)

	var k8sPDEscalationPolicy *v1alpha1.EscalationPolicy

	pd_client := pagerduty.NewClient("")

	adapter := EPAdapter{
		PD_Client: pd_client,
	}

	var policyID string

	Context("CRUD operations on Policy", func() {
		BeforeEach(func() {
			k8sPDEscalationPolicy = &v1alpha1.EscalationPolicy{
				Spec: v1alpha1.EscalationPolicySpec{
					Name:                       PolicyName,
					Description:                PolicyDescription,
					NumLoops:                   NumLoops,
					OnCallHandoffNotifications: OnCallHandoffNotifications,
					EscalationRules: []typeinfo.K8sEscalationRule{
						{
							Targets: []typeinfo.UserID{
								typeinfo.UserID(MockUserID),
							},
							Delay: Delay,
						},
					},
				},
			}
		})

		AfterEach(func() {
			if policyID != "" {
				pd_client.DeleteEscalationPolicyWithContext(context.TODO(), policyID)
			}
		})

		Describe("Creating policies", func() {
			Context("With correct fields", func() {
				It("should create a policy upstream", func() {
					policyId, err := adapter.CreateEscalationPolicy(&k8sPDEscalationPolicy.Spec)
					Expect(err).NotTo(HaveOccurred())
					Expect(policyId).NotTo(Equal(""))

					pdPolicy, err := pd_client.GetEscalationPolicyWithContext(
						context.TODO(),
						policyId,
						&pagerduty.GetEscalationPolicyOptions{},
					)
					Expect(err).NotTo(HaveOccurred())

					compareLocalToUpstream(pdPolicy, k8sPDEscalationPolicy)
				})
			})
		})

		Describe("Updating policies", func() {
			Context("With correct fields", func() {
				It("should create a policy upstream", func() {
					policyId, err := adapter.CreateEscalationPolicy(&k8sPDEscalationPolicy.Spec)
					Expect(err).NotTo(HaveOccurred())
					Expect(policyId).NotTo(Equal(""))

					newName := "Policy NewName"

					k8sPDEscalationPolicy.Status.PolicyID = policyId
					k8sPDEscalationPolicy.Spec.Name = newName

					err = adapter.UpdatePDEscalationPolicy(k8sPDEscalationPolicy)

					Expect(err).NotTo(HaveOccurred())

					pdPolicy, err := pd_client.GetEscalationPolicyWithContext(
						context.TODO(),
						policyId,
						&pagerduty.GetEscalationPolicyOptions{},
					)
					Expect(err).NotTo(HaveOccurred())

					compareLocalToUpstream(pdPolicy, k8sPDEscalationPolicy)
				})
			})
		})

		Describe("Deleting policies", func() {
			Context("With correct fields", func() {
				It("should delete a policy upstream", func() {
					policyId, err := adapter.CreateEscalationPolicy(&k8sPDEscalationPolicy.Spec)
					Expect(err).NotTo(HaveOccurred())
					Expect(policyId).NotTo(Equal(""))

					err = adapter.DeletePDEscalationPolicy(policyId)
					Expect(err).NotTo(HaveOccurred())

					pdPolicy, err := pd_client.GetEscalationPolicyWithContext(
						context.TODO(),
						policyId,
						&pagerduty.GetEscalationPolicyOptions{},
					)
					Expect(pdPolicy).To(BeNil())
					Expect(err).To(HaveOccurred())
				})
			})
		})

	})
})
