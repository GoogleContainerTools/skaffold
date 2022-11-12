package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"

	_ "embed"

	"cloud.google.com/go/spanner"
	database "cloud.google.com/go/spanner/admin/database/apiv1"
	instance "cloud.google.com/go/spanner/admin/instance/apiv1"
	adminpb "google.golang.org/genproto/googleapis/spanner/admin/database/v1"
	databasepb "google.golang.org/genproto/googleapis/spanner/admin/database/v1"
	instancepb "google.golang.org/genproto/googleapis/spanner/admin/instance/v1"
	"google.golang.org/grpc/codes"
)

// CreateInstance creates an instance in spanner parsed by a DB url.
func CreateInstance(ctx context.Context, iac *instance.InstanceAdminClient, uri string) error {
	matches := regexp.MustCompile("projects/(.*)/instances/(.*)/databases/.*").FindStringSubmatch(uri)
	if matches == nil || len(matches) != 3 {
		fmt.Printf("invalid instance id %s", uri)
		return fmt.Errorf("invalid instance id %s", uri)
	}
	instanceName := "projects/" + matches[1] + "/instances/" + matches[2]

	_, err := iac.GetInstance(ctx, &instancepb.GetInstanceRequest{
		Name: instanceName,
	})
	if err != nil && spanner.ErrCode(err) != codes.NotFound {
		return err
	}
	if err == nil {
		// instance already exists
		return nil
	}
	_, err = iac.CreateInstance(ctx, &instancepb.CreateInstanceRequest{
		Parent:     "projects/" + matches[1],
		InstanceId: matches[2],
	})

	log.Printf("Created instance [%s]\n", matches[2])
	if err != nil {
		return err
	}
	return nil
}

// CreateDatabase creates a DB in the spanner. Instance must exist for a DB URL. If drop boolean flag is set then it drops
// DB if it exists.
func CreateDatabase(ctx context.Context, dac *database.DatabaseAdminClient, uri string, drop bool) error {
	matches := regexp.MustCompile("^(.*)/databases/(.*)$").FindStringSubmatch(uri)
	if matches == nil || len(matches) != 3 {
		log.Printf("invalid database id %s", uri)
		return fmt.Errorf("invalid database id %s", uri)
	}
	_, err := dac.GetDatabase(ctx, &databasepb.GetDatabaseRequest{Name: uri})
	if err != nil && spanner.ErrCode(err) != codes.NotFound {
		return err
	}
	if err == nil {
		// Database already exists
		if drop {
			if err = dac.DropDatabase(ctx, &databasepb.DropDatabaseRequest{Database: uri}); err != nil {
				fmt.Println(err)
				return err
			}
		} else {
			return nil
		}
	}

	op, err := dac.CreateDatabase(ctx, &databasepb.CreateDatabaseRequest{
		Parent:          matches[1],
		CreateStatement: "CREATE DATABASE `" + matches[2] + "`",
		ExtraStatements: []string{},
	})
	if err != nil {
		return err
	}
	if _, err = op.Wait(ctx); err != nil {
		return err
	}
	return nil
}

// ApplyDDL apply a series of DDL statements(For ex: Create table, index etc.) on a database instance.
func ApplyDDL(adminClient *database.DatabaseAdminClient, dbName string, ddl string) error {
	scanner := bufio.NewScanner(strings.NewReader(ddl))
	scanner.Split(scanDDL)
	for scanner.Scan() {
		err := updateDDL(adminClient, dbName, scanner.Text())
		if err != nil {
			return err
		}
	}
	return nil
}

func updateDDL(adminClient *database.DatabaseAdminClient, dbName string, statements ...string) error {
	ctx := context.Background()
	log.Printf("Database  name: %q", dbName)
	log.Printf("DDL update: %q", statements)
	op, err := adminClient.UpdateDatabaseDdl(ctx, &adminpb.UpdateDatabaseDdlRequest{
		Database:   dbName,
		Statements: statements,
	})
	if err != nil {
		return err
	}
	err = op.Wait(ctx)
	if err != nil {
		log.Printf("Got error as: %q", err)
		return err
	}
	log.Println("Table  created: ")
	return nil
}

func scanDDL(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) <= 1 {
		return 0, nil, nil
	}
	if i := bytes.Index(data, []byte(";")); i >= 0 {
		fmt.Print(i)
		return i + 1, data[0:i], nil
	}
	if atEOF {
		return len(data), data, nil
	}
	return 0, nil, nil
}
