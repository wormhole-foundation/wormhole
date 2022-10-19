module github.com/certusone/wormhole/event_database/functions_server

go 1.16

// cloud runtime is go 1.16. just for reference.

require (
	cloud.google.com/go/pubsub v1.17.1
	github.com/GoogleCloudPlatform/functions-framework-go v1.5.2
	github.com/certusone/wormhole/event_database/cloud_functions v0.0.0-20220126152252-d4735fc7c1aa
)

replace (
	github.com/btcsuite/btcd => github.com/btcsuite/btcd v0.23.0
	github.com/certusone/wormhole/event_database/cloud_functions => ../cloud_functions
	github.com/gogo/protobuf => github.com/regen-network/protobuf v1.3.3-alpha.regen.1
)
