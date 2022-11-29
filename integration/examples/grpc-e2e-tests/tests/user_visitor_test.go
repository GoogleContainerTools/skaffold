package test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	pb "github.com/GoogleContainerTools/skaffold/v2/integration/examples/grpc-e2e-tests/service/proto"
	"github.com/google/uuid"
)

var _ = Describe("Visit and Gets visit count for an user", func() {
	ctx := context.Background()
	client := CreateVisitorServiceClient()
	userId := uuid.New().String()
	Context("Visit for first time", func() {
		request := &pb.UpdateVisitorRequest{Visitor: &pb.Visitor{Name: userId}}

		_, err := client.UpdateVisitor(ctx, request)
		It("should not throw error", func() {
			Expect(err).To(BeNil())
		})

	})
	Context("GetVisit count", func() {
		request := &pb.VisitorCounterRequest{Visitor: &pb.Visitor{Name: userId}}
		response, _ := client.GetVisitCount(ctx, request)
		It("visit count should be 1", func() {
			Expect(response.Count).Should(Equal(int64(1)))
		})

	})
})
