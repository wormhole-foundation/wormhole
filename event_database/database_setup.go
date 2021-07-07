package main

import (
	"context"
	"log"

	"cloud.google.com/go/bigtable"
	"google.golang.org/api/option"
)

// sliceContains reports whether the provided string is present in the given slice of strings.
func sliceContains(list []string, target string) bool {
	for _, s := range list {
		if s == target {
			return true
		}
	}
	return false
}

// RunSetup will create a table and column families, if they do not already exist.
func RunSetup(project string, instance string, keyFilePath string) {

	ctx := context.Background()

	// Set up admin client, tables, and column families.
	adminClient, err := bigtable.NewAdminClient(ctx, project, instance, option.WithCredentialsFile(keyFilePath))
	if err != nil {
		log.Fatalf("Could not create admin client: %v", err)
	}

	tables, err := adminClient.Tables(ctx)
	if err != nil {
		log.Fatalf("Could not fetch table list: %v", err)
	}

	if !sliceContains(tables, tableName) {
		log.Printf("Creating table %s", tableName)
		if err := adminClient.CreateTable(ctx, tableName); err != nil {
			log.Fatalf("Could not create table %s: %v", tableName, err)
		}
	}

	tblInfo, err := adminClient.TableInfo(ctx, tableName)
	if err != nil {
		log.Fatalf("Could not read info for table %s: %v", tableName, err)
	}

	for _, familyName := range columnFamilies {
		if !sliceContains(tblInfo.Families, familyName) {
			if err := adminClient.CreateColumnFamily(ctx, tableName, familyName); err != nil {
				log.Fatalf("Could not create column family %s: %v", familyName, err)
			}
		}
	}

	if err = adminClient.Close(); err != nil {
		log.Fatalf("Could not close admin client: %v", err)
	}
}
