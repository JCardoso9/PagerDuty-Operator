package pdservice

import (
	"context"

	"github.com/PagerDuty/go-pagerduty"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gitlab.share-now.com/platform/pagerduty-operator/api/v1alpha1"
)

func compareLocalToUpstream(k8sPDService *v1alpha1.PagerdutyService, pdService *pagerduty.Service) {
	GinkgoHelper()
	Expect(pdService.Name).To(Equal(k8sPDService.Spec.Name))
	Expect(pdService.Description).To(Equal(k8sPDService.Spec.Description))
	Expect(pdService.AutoResolveTimeout).To(Equal(k8sPDService.Spec.AutoResolveTimeout))
	Expect(pdService.AcknowledgementTimeout).To(Equal(k8sPDService.Spec.AcknowledgementTimeout))
	Expect(pdService.Status).To(Equal(k8sPDService.Spec.Status))
	Expect(pdService.AlertCreation).To(Equal(k8sPDService.Spec.AlertCreation))
}

var _ = Describe("PagerDuty Service adapter tests", func() {

	const (
		PDServiceName        string = "Test-Service"
		PDServiceDescription string = "Test-Service-description"
		Status               string = "active"
		AlertCreation        string = "create_incidents"
		EscalationPolicyName string = "test-policy"
		EscalationPolicyID   string = "PJLKMZK" // Already created manually
	)

	var AutoResolveTimeout uint = 14400
	var AcknowledgementTimeout uint = 1800

	var k8sPDService *v1alpha1.PagerdutyService

	pd_client := pagerduty.NewClient("")

	adapter := PDServiceAdapter{
		PD_Client: pd_client,
	}

	var serviceID string

	Context("CRUD operations on Service", func() {
		BeforeEach(func() {
			k8sPDService = &v1alpha1.PagerdutyService{
				Spec: v1alpha1.PagerdutyServiceSpec{
					Name:                   PDServiceName,
					Description:            PDServiceDescription,
					AutoResolveTimeout:     &AutoResolveTimeout,
					AcknowledgementTimeout: &AcknowledgementTimeout,
					Status:                 Status,
					EscalationPolicyName:   EscalationPolicyName,
					AlertCreation:          AlertCreation,
				},
				Status: v1alpha1.PagerdutyServiceStatus{
					ServiceID: "PJLKMZK",
				},
			}
		})

		AfterEach(func() {
			if serviceID != "" {
				pd_client.DeleteServiceWithContext(context.TODO(), serviceID)
			}
		})

		Describe("Creating business services", func() {
			Context("With correct fields", func() {
				It("should create a policy upstream", func() {
					serviceId, err := adapter.CreatePDService(k8sPDService)
					Expect(err).NotTo(HaveOccurred())
					Expect(serviceId).NotTo(Equal(""))

					pdService, err := pd_client.GetServiceWithContext(
						context.TODO(),
						serviceId,
						&pagerduty.GetServiceOptions{},
					)
					Expect(err).NotTo(HaveOccurred())

					compareLocalToUpstream(k8sPDService, pdService)
				})
			})
		})

		Describe("Updating services", func() {
			Context("With correct fields", func() {
				It("should create a service upstream", func() {
					serviceId, err := adapter.CreatePDService(k8sPDService)
					Expect(err).NotTo(HaveOccurred())
					Expect(serviceId).NotTo(Equal(""))

					newName := "Service NewName"

					k8sPDService.Status.ServiceID = serviceId
					k8sPDService.Spec.Name = newName

					err = adapter.UpdatePDService(k8sPDService)
					Expect(err).NotTo(HaveOccurred())

					pdService, err := pd_client.GetServiceWithContext(
						context.TODO(),
						serviceId,
						&pagerduty.GetServiceOptions{},
					)
					Expect(err).NotTo(HaveOccurred())

					compareLocalToUpstream(k8sPDService, pdService)
				})
			})
		})

		Describe("Deleting business services", func() {
			Context("With correct fields", func() {
				It("should delete a policy upstream", func() {
					serviceId, err := adapter.CreatePDService(k8sPDService)
					Expect(err).NotTo(HaveOccurred())
					Expect(serviceId).NotTo(Equal(""))

					err = adapter.DeletePDService(serviceId)
					Expect(err).NotTo(HaveOccurred())

					pdService, err := pd_client.GetServiceWithContext(
						context.TODO(),
						serviceId,
						&pagerduty.GetServiceOptions{},
					)
					Expect(pdService).To(BeNil())
					Expect(err).To(HaveOccurred())
				})
			})
		})
	})
})
