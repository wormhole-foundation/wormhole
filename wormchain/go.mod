module github.com/wormhole-foundation/wormchain

go 1.22.5

require (
	cosmossdk.io/api v0.7.5
	cosmossdk.io/errors v1.0.1
	cosmossdk.io/log v1.3.1
	cosmossdk.io/math v1.3.0
	cosmossdk.io/simapp v0.0.0-20240716145229-670881847082
	cosmossdk.io/tools/rosetta v0.2.1
	github.com/CosmWasm/wasmd v0.45.0
	github.com/CosmWasm/wasmvm v1.5.2
	github.com/cometbft/cometbft v1.0.0-rc1
	github.com/cometbft/cometbft-db v0.12.0
	github.com/cosmos/cosmos-sdk v0.51.0
	github.com/cosmos/gogoproto v1.5.0
	github.com/cosmos/ibc-apps/middleware/packet-forward-middleware/v7 v7.1.3
	github.com/cosmos/ibc-go/v7 v7.6.0
	github.com/ethereum/go-ethereum v1.10.21
	github.com/golang/protobuf v1.5.4
	github.com/gorilla/mux v1.8.1
	github.com/grpc-ecosystem/grpc-gateway v1.16.0
	github.com/holiman/uint256 v1.2.1
	github.com/prometheus/client_golang v1.19.1
	github.com/spf13/cast v1.6.0
	github.com/spf13/cobra v1.8.1
	github.com/stretchr/testify v1.9.0
	github.com/wormhole-foundation/wormhole/sdk v0.0.0-20220926172624-4b38dc650bb0
	google.golang.org/genproto/googleapis/api v0.0.0-20240617180043-68d350f18fd4
	google.golang.org/grpc v1.64.1
)

require (
	github.com/google/go-cmp v0.6.0 // indirect
	golang.org/x/exp v0.0.0-20240531132922-fd00a4e0eefc // indirect
	google.golang.org/protobuf v1.34.2 // indirect
)

replace (
	// cosmos keyring
	github.com/99designs/keyring => github.com/cosmos/keyring v1.2.0
	// wormhole forks
	// github.com/CosmWasm/wasmd v0.45.0 => github.com/wormhole-foundation/wasmd v0.30.0-wormchain-2
	github.com/cosmos/cosmos-sdk => ../../wh-cosmos-sdk
	github.com/gogo/protobuf => github.com/regen-network/protobuf v1.3.3-alpha.regen.1

	// v0.47.0 changelog replace statements
	github.com/syndtr/goleveldb => github.com/syndtr/goleveldb v1.0.1-0.20210819022825-2ae1ddf74ef7
	github.com/wormhole-foundation/wormhole/sdk => ../sdk
	golang.org/x/exp => golang.org/x/exp v0.0.0-20230711153332-06a737ee72cb
)
