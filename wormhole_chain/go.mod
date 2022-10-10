module github.com/wormhole-foundation/wormhole-chain

go 1.16

require (
	github.com/cosmos/cosmos-sdk v0.45.7
	github.com/cosmos/ibc-go v1.2.2
	github.com/dgraph-io/ristretto v0.1.0 // indirect
	github.com/ethereum/go-ethereum v1.10.21
	github.com/gogo/protobuf v1.3.3
	github.com/golang/glog v1.0.0 // indirect
	github.com/golang/protobuf v1.5.2
	github.com/google/btree v1.0.1 // indirect
	github.com/gorilla/mux v1.8.0
	github.com/grpc-ecosystem/grpc-gateway v1.16.0
	github.com/spf13/cast v1.5.0
	github.com/spf13/cobra v1.5.0
	github.com/stretchr/testify v1.8.0
	github.com/tendermint/spm v0.1.9
	github.com/tendermint/tendermint v0.34.20
	github.com/tendermint/tm-db v0.6.7
	github.com/wormhole-foundation/wormhole/sdk v0.0.0-20220921004715-3103e59217da
	google.golang.org/genproto v0.0.0-20220519153652-3a47de7e79bd
	google.golang.org/grpc v1.48.0
	nhooyr.io/websocket v1.8.7 // indirect
)

replace (
	github.com/99designs/keyring => github.com/cosmos/keyring v1.1.7-0.20210622111912-ef00f8ac3d76
	github.com/cosmos/cosmos-sdk v0.45.7 => github.com/wormhole-foundation/cosmos-sdk v0.45.7-wormhole
	github.com/gogo/protobuf => github.com/regen-network/protobuf v1.3.3-alpha.regen.1
	github.com/wormhole-foundation/wormhole/sdk => ../sdk
	google.golang.org/grpc => google.golang.org/grpc v1.33.2
)
