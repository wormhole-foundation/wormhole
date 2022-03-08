package main

import (
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

func main() {
	fmt.Println("vim-go")

	if len(os.Args) != 2 {
    fmt.Fprintf(os.Stderr, "Bucket name required\nUsage: %s network ['testnet']\n", os.Args[0])
    os.Exit(1)
	}

	network := os.Args[1]
	bucket := "algorand-repository"


	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("us-east-1")},
	)

	// Create S3 service client
	svc := s3.New(sess)

	prefix := fmt.Sprintf("%s/ledger-snapshots/", network)
	resp, err := svc.ListObjectsV2(&s3.ListObjectsV2Input{Bucket: aws.String(bucket), Prefix: aws.String(prefix)})
	if err != nil {
    fmt.Fprintf(os.Stderr, "Unable to list items in bucket %q, %v\n", bucket, err)
    os.Exit(1)
	}

	for _, item := range resp.Contents {
		// testnet/ledger-snapshots/2019-11-22T21:00:42.146Z
		fmt.Println("Name:         ", *item.Key)
		fmt.Println("Last modified:", *item.LastModified)
		fmt.Println("Size:         ", *item.Size)
		fmt.Println("Storage class:", *item.StorageClass)
		fmt.Println("")
	}
}
