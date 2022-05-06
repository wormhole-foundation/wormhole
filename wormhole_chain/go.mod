module github.com/certusone/wormhole-chain

go 1.16

require (
	github.com/certusone/wormhole/node v0.0.0-20211115153408-0a93202f6e5d
	github.com/cosmos/cosmos-sdk v0.44.1
	github.com/cosmos/ibc-go v1.2.0
	github.com/ethereum/go-ethereum v1.10.6
	github.com/gogo/protobuf v1.3.3
	github.com/golang/glog v1.0.0 // indirect
	github.com/golang/protobuf v1.5.2
	github.com/google/go-cmp v0.5.7 // indirect
	github.com/gorilla/mux v1.8.0
	github.com/grpc-ecosystem/grpc-gateway v1.16.0
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.10.0 // indirect
	github.com/holiman/uint256 v1.2.0
	github.com/regen-network/cosmos-proto v0.3.1 // indirect
	github.com/spf13/cast v1.3.1
	github.com/spf13/cobra v1.2.1
	github.com/stretchr/testify v1.7.0
	github.com/tendermint/spm v0.1.5
	github.com/tendermint/tendermint v0.34.13
	github.com/tendermint/tm-db v0.6.4
	golang.org/x/net v0.0.0-20220127200216-cd36cc0744dd // indirect
	google.golang.org/genproto v0.0.0-20220317150908-0efb43f6373e
	google.golang.org/grpc v1.45.0
)

replace (
	github.com/99designs/keyring => github.com/cosmos/keyring v1.1.7-0.20210622111912-ef00f8ac3d76
	github.com/cosmos/cosmos-sdk => ./cosmos-sdk
	github.com/gogo/protobuf => github.com/regen-network/protobuf v1.3.3-alpha.regen.1
	google.golang.org/grpc => google.golang.org/grpc v1.33.2
)
