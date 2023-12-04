package business_service

import (
	"context"

	"github.com/PagerDuty/go-pagerduty"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gitlab.share-now.com/platform/pagerduty-operator/api/v1alpha1"
)

func compareLocalToUpstream(k8sBusService *v1alpha1.BusinessService, pdBusService *pagerduty.BusinessService) {
	GinkgoHelper()
	Expect(pdBusService.Name).To(Equal(k8sBusService.Spec.Name))
	Expect(pdBusService.Description).To(Equal(k8sBusService.Spec.Description))
	Expect(pdBusService.PointOfContact).To(Equal(k8sBusService.Spec.PointOfContact))
	Expect(pdBusService.Team.ID).To(Equal(k8sBusService.Spec.TeamID))
}

var _ = Describe("Business service adapter tests", func() {

	const (
		BusServiceName        string = "Test-Business Service"
		BusServiceDescription string = "Test-Business Service-description"
		PointOfContact        string = "Username"
		TeamID                string = "MOCKTEAMID"
	)

	var k8sBusService *v1alpha1.BusinessService

	pd_client := pagerduty.NewClient("")

	adapter := BSAdapter{
		PD_Client: pd_client,
	}

	var busServiceID string

	Context("CRUD operations on Business Service", func() {
		BeforeEach(func() {
			k8sBusService = &v1alpha1.BusinessService{
				Spec: v1alpha1.BusinessServiceSpec{
					Name:           BusServiceName,
					Description:    BusServiceDescription,
					PointOfContact: PointOfContact,
					TeamID:         TeamID,
				},
			}
		})

		AfterEach(func() {
			if busServiceID != "" {
				pd_client.DeleteBusinessServiceWithContext(context.TODO(), busServiceID)
			}
		})

		Describe("Creating business services", func() {
			Context("With correct fields", func() {
				It("should create a business service upstream", func() {
					busServiceId, err := adapter.CreateBusinessService(&k8sBusService.Spec)
					Expect(err).NotTo(HaveOccurred())
					Expect(busServiceId).NotTo(Equal(""))

					pdBusService, err := pd_client.GetBusinessServiceWithContext(
						context.TODO(),
						busServiceId,
					)
					Expect(err).NotTo(HaveOccurred())

					compareLocalToUpstream(k8sBusService, pdBusService)
				})
			})
		})

		Describe("Updating business services", func() {
			Context("With correct fields", func() {
				It("should create a business service upstream", func() {
					busServiceId, err := adapter.CreateBusinessService(&k8sBusService.Spec)
					Expect(err).NotTo(HaveOccurred())
					Expect(busServiceId).NotTo(Equal(""))

					newName := "Business Service NewName"

					k8sBusService.Status.BusinessServiceID = busServiceId
					k8sBusService.Spec.Name = newName

					err = adapter.UpdateBusinessService(k8sBusService)
					Expect(err).NotTo(HaveOccurred())

					pdBusService, err := pd_client.GetBusinessServiceWithContext(
						context.TODO(),
						busServiceId,
					)
					Expect(err).NotTo(HaveOccurred())

					compareLocalToUpstream(k8sBusService, pdBusService)
				})
			})
		})

		Describe("Deleting business services", func() {
			Context("With correct fields", func() {
				It("should delete a business service upstream", func() {
					busServiceId, err := adapter.CreateBusinessService(&k8sBusService.Spec)
					Expect(err).NotTo(HaveOccurred())
					Expect(busServiceId).NotTo(Equal(""))

					err = adapter.DeleteBusinessService(busServiceId)
					Expect(err).NotTo(HaveOccurred())

					pdBusService, err := pd_client.GetBusinessServiceWithContext(
						context.TODO(),
						busServiceId,
					)
					Expect(pdBusService).To(BeNil())
					Expect(err).To(HaveOccurred())
				})
			})
		})

	})
})
