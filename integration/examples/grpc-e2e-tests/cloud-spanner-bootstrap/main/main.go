package main

import (
	"context"
	"log"
	"os"

	database "cloud.google.com/go/spanner/admin/database/apiv1"
	instance "cloud.google.com/go/spanner/admin/instance/apiv1"
	"google.golang.org/api/option"
	"google.golang.org/grpc"

	_ "embed"
)

//go:embed visitor_table.ddl
var visitorTableSchema string

func main() {
	ctx := context.Background()
	spannerHost := os.Getenv("SPANNER_EMULATOR_HOST")
	if spannerHost == "" {
		log.Fatal("SPANNER_EMULATOR_HOST environment variable not set")
	}
	db := os.Getenv("DATABASE")
	if db == "" {
		log.Fatal("DATABASE environment variable not set")
	}
	iac, err := instance.NewInstanceAdminClient(ctx, option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithInsecure()),
		option.WithEndpoint(spannerHost))
	if err != nil {
		log.Fatal("Error while creating spanner instance admin client", err)
	}
	defer iac.Close()

	err = CreateInstance(ctx, iac, db)
	if err != nil {
		log.Fatal("Error while creating spanner instance", err)
	}
	dac, err := database.NewDatabaseAdminClient(ctx, option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithInsecure()),
		option.WithEndpoint(spannerHost))

	if err != nil {
		log.Fatal("Error while creating spanner DB client", err)
	}
	defer dac.Close()

	ddl := visitorTableSchema
	err = CreateDatabase(ctx, dac, db, true)
	if err != nil {
		log.Fatal("Error while creating spanner DB", err)
	}
	err = ApplyDDL(dac, db, ddl)
	if err != nil {
		log.Fatal("Failed to apply DDL into spanner DB", err)
	}
}
