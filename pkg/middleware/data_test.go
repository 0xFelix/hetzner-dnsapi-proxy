package middleware // Keep the original package to access unexported symbols like reqDataKey

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Context Data Handling", func() {

	Describe("newContextWithReqData", func() {
		It("should store ReqData in the context", func() {
			ctx := context.Background()
			expectedData := &ReqData{
				FullName: "test.example.com",
				Name:     "test",
				Zone:     "example.com",
			}

			newCtx := newContextWithReqData(ctx, expectedData)

			Expect(newCtx).NotTo(BeIdenticalTo(ctx), "newContextWithReqData should return a new context")

			// Retrieve the data using the package's specific key to verify
			// This line requires the test to be in `package middleware` to access reqDataKey
			retrievedData, ok := newCtx.Value(reqDataKey).(*ReqData)
			Expect(ok).To(BeTrue(), "ReqData should be found in context using reqDataKey")
			Expect(retrievedData).To(BeIdenticalTo(expectedData), "Retrieved data pointer should match expected data pointer")
			Expect(retrievedData.FullName).To(Equal(expectedData.FullName))
		})
	})

	Describe("reqDataFromContext", func() {
		Context("when ReqData is present in the context", func() {
			It("should retrieve the ReqData and no error", func() {
				originalCtx := context.Background()
				expectedData := &ReqData{
					FullName: "sub.domain.org",
					Value:    "123.123.123.123",
				}

				ctxWithData := newContextWithReqData(originalCtx, expectedData)

				retrievedData, err := reqDataFromContext(ctxWithData)

				Expect(err).NotTo(HaveOccurred())
				Expect(retrievedData).NotTo(BeNil())
				Expect(retrievedData).To(BeIdenticalTo(expectedData), "Retrieved data pointer should match expected data pointer")
				Expect(retrievedData.FullName).To(Equal(expectedData.FullName))
				Expect(retrievedData.Value).To(Equal(expectedData.Value))
			})
		})

		Context("when ReqData is not in the context", func() {
			It("should return an error and nil data", func() {
				ctxWithoutData := context.Background() // Plain context

				retrievedData, err := reqDataFromContext(ctxWithoutData)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("ReqData not found in context"))
				Expect(retrievedData).To(BeNil())
			})
		})
	})
})
