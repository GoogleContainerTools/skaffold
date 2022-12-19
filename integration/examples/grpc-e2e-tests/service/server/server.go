package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"

	"google.golang.org/grpc"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"

	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"

	pb "github.com/GoogleContainerTools/skaffold/v2/integration/examples/grpc-e2e-tests/service/proto"
)

var (
	port = flag.Int("port", 8080, "The server port")
)

const (
	visitorTableName = "VisitorCounter"
)

var visitorColumns = []string{"UserName", "VisitCount"}

func main() {
	ctx := context.Background()
	flag.Parse()
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	db := os.Getenv("DATABASE")
	if db == "" {
		log.Fatal("DATABASE environment variable not set")
	}

	spannerClient, err := spanner.NewClient(ctx, db)
	if err != nil {
		log.Fatal("Error while initializing the spanner client")
	}
	grpcServer := grpc.NewServer(
		grpc.ConnectionTimeout(time.Second),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle: time.Second * 10,
			Timeout:           time.Second * 20,
		}),
		grpc.KeepaliveEnforcementPolicy(
			keepalive.EnforcementPolicy{
				MinTime:             time.Second,
				PermitWithoutStream: true,
			}),
		grpc.MaxConcurrentStreams(5),
	)
	log.Println("Started visitor server")
	pb.RegisterVisitorCounterServer(grpcServer, &server{SpannerClient: spannerClient})
	reflection.Register(grpcServer)
	go func(grpcServer *grpc.Server) {
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}(grpcServer)
	<-make(chan int)

}

func (s *server) UpdateVisitor(ctx context.Context, in *pb.UpdateVisitorRequest) (*pb.UpdateVisitorResponse, error) {
	log.Printf("Got request %v", in)
	visitor := in.GetVisitor()
	visitCount := int64(0)
	existingVisitorRecord, err := s.GetVisitCount(ctx, &pb.VisitorCounterRequest{Visitor: visitor})
	if existingVisitorRecord != nil {
		log.Printf("Got existing count as %v", existingVisitorRecord)
		visitCount = existingVisitorRecord.Count
	}
	// Update visit count by 1.
	visitCount++
	m := []*spanner.Mutation{
		spanner.InsertOrUpdate(visitorTableName, visitorColumns, []interface{}{visitor.GetName(), visitCount}),
	}
	_, err = s.SpannerClient.Apply(ctx, m)
	if err != nil {
		return nil, err
	}
	return &pb.UpdateVisitorResponse{}, nil
}

func (s *server) GetVisitCount(ctx context.Context, in *pb.VisitorCounterRequest) (*pb.VisitorCounterResponse, error) {
	log.Printf("Got request %v", in)
	stmt := spanner.NewStatement(fmt.Sprintf(`
				SELECT %s FROM %s  WHERE UserName=@userName`,
		strings.Join(visitorColumns, ", "), visitorTableName))
	stmt.Params["userName"] = in.GetVisitor().GetName()
	iter := s.SpannerClient.Single().Query(ctx, stmt)
	defer iter.Stop()

	for {
		row, err := iter.Next()
		if err == iterator.Done {
			fmt.Println("Do not get any records")
			return &pb.VisitorCounterResponse{Visitor: &pb.Visitor{Name: in.GetVisitor().GetName()}, Count: 0}, nil
		}
		if err != nil {
			return nil, err
		}
		var count int64
		var name string
		if err := row.Columns(&name, &count); err != nil {
			return nil, err
		}

		return &pb.VisitorCounterResponse{Visitor: &pb.Visitor{Name: name}, Count: count}, nil

	}
	return &pb.VisitorCounterResponse{Visitor: &pb.Visitor{Name: in.GetVisitor().GetName()}, Count: 0}, nil
}

type server struct {
	SpannerClient *spanner.Client
	pb.UnimplementedVisitorCounterServer
}
