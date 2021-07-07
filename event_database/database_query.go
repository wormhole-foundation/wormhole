package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"log"
	"time"

	"cloud.google.com/go/bigtable"
	"google.golang.org/api/option"
)

type Signature struct {
	// Index of the validator
	Index uint8
	// Signature data
	Signature [65]byte
}

func printRow(row bigtable.Row) {
	if _, ok := row[columnFamilies[0]]; ok {
		printItems(row[columnFamilies[0]])
	}
	if _, ok := row[columnFamilies[1]]; ok {
		printItems(row[columnFamilies[1]])
	}
	if _, ok := row[columnFamilies[2]]; ok {
		printSignatures(row[columnFamilies[2]])
	}
	if _, ok := row[columnFamilies[3]]; ok {
		printItems(row[columnFamilies[3]])
	}
}

func printItems(familyCols []bigtable.ReadItem) {
	for _, item := range familyCols {

		log.Printf("\t%s = %s\n", item.Column, string(item.Value))
	}
}

func printSignatures(familyCols []bigtable.ReadItem) {
	for _, item := range familyCols {
		if item.Column == "VAAState:Signatures" {
			reader := bytes.NewReader(item.Value[:])
			lenSignatures, er := reader.ReadByte()
			if er != nil {
				log.Print(fmt.Errorf("failed to read signature length"))
				return
			}

			signatures := make([]*Signature, lenSignatures)
			for i := 0; i < int(lenSignatures); i++ {
				index, err := reader.ReadByte()
				if err != nil {
					log.Print(fmt.Errorf("failed to read validator index [%d]", i))
					return
				}

				signature := [65]byte{}
				if n, err := reader.Read(signature[:]); err != nil || n != 65 {
					log.Print(fmt.Errorf("failed to read signature [%d]: %w", i, err))
					return
				}

				signatures[i] = &Signature{
					Index:     index,
					Signature: signature,
				}
			}
			for index, sig := range signatures {
				log.Printf("\tSignatures: list index: %v, item index: %v, signature = %s\n", index, sig.Index, hex.EncodeToString(sig.Signature[:]))
			}
		} else {
			log.Printf("\t%s = %s\n", item.Column, string(item.Value))
		}
	}
}

// Query will lookup BigTable row(s) and log their data.
func Query(project string, instance string, keyFilePath string, rowKey string, previousMinutes int) {

	ctx := context.Background()

	client, err := bigtable.NewClient(ctx, project, instance, option.WithCredentialsFile(keyFilePath))
	if err != nil {
		log.Fatalf("Could not create data operations client: %v", err)
	}

	tbl := client.Open(tableName)

	if rowKey != "" {
		log.Printf("Querying by row key: %s ", rowKey)
		row, err := tbl.ReadRow(ctx, rowKey)
		if err != nil {
			log.Fatalf("Could not read row with key %s: %v", rowKey, err)
		}

		printRow(row)
	}

	if previousMinutes != 0 {
		xMinutesAgo := time.Now().Add(-time.Duration(previousMinutes) * time.Minute)
		log.Printf("Reading rows from: %v ", xMinutesAgo)

		err = tbl.ReadRows(ctx, bigtable.PrefixRange(""), func(row bigtable.Row) bool {
			if _, ok := row[columnFamilies[0]]; ok {
				// log the rowKey
				cell := row[columnFamilies[0]][0]
				log.Printf("rowKey: %s", cell.Row)
			}
			printRow(row)
			return true
		}, bigtable.RowFilter(bigtable.TimestampRangeFilter(xMinutesAgo, time.Now())))
		if err != nil {
			log.Fatalf("failed to read recent rows: %v", err)
		}
	}

	if err = client.Close(); err != nil {
		log.Fatalf("Could not close data operations client: %v", err)
	}
}
