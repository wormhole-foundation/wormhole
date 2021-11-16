module github.com/certusone/wormhole/event_database/cloud_functions

go 1.16

// cloud runtime is go 1.16. just for reference.

require (
	cloud.google.com/go/bigtable v1.10.1
	cloud.google.com/go/pubsub v1.3.1
	github.com/GoogleCloudPlatform/functions-framework-go v1.3.0
	github.com/certusone/wormhole/node v0.0.0-20211115153408-0a93202f6e5d
	github.com/cosmos/cosmos-sdk v0.44.0
	github.com/gagliardetto/solana-go v1.0.2
	github.com/holiman/uint256 v1.2.0
	github.com/mattn/go-isatty v0.0.14 // indirect
	golang.org/x/net v0.0.0-20210903162142-ad29c8ab022f // indirect
	golang.org/x/sys v0.0.0-20210903071746-97244b99971b // indirect
	google.golang.org/api v0.48.0 // indirect
	google.golang.org/grpc v1.40.0 // indirect
	google.golang.org/protobuf v1.27.1 // indirect
)

replace github.com/gogo/protobuf => github.com/regen-network/protobuf v1.3.3-alpha.regen.1
