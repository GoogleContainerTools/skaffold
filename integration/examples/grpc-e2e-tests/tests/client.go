package test

import (
	"log"
	"net/url"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/GoogleContainerTools/skaffold/v2/examples/grpc-e2e-tests/service/proto"
)

// CreateVisitorServiceClient returns VisitorServiceClient which connects to
// visitor counter server.
func CreateVisitorServiceClient() pb.VisitorCounterClient {
	// Address of sensor project service. The same setup can be
	// used against staging as long as the SENSOR_PROJECT_SERVICE env
	// variable is set to staging service address.
	visitorApiAddr := os.Getenv("VISITOR_COUNTER_SERVICE")
	u, err := url.ParseRequestURI(visitorApiAddr)
	conn, err := grpc.Dial(u.Host, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Could not connect to local sensor project server : %v", err)
	}
	client := pb.NewVisitorCounterClient(conn)
	return client
}
