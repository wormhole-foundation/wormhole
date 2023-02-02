module github.com/wormhole-foundation/wormchain

go 1.16

require (
	github.com/CosmWasm/wasmd v0.30.0
	github.com/CosmWasm/wasmvm v1.1.1
	github.com/cosmos/cosmos-sdk v0.45.11
	github.com/cosmos/ibc-go/v3 v3.3.0
	github.com/cosmos/ibc-go/v4 v4.2.0
	github.com/ethereum/go-ethereum v1.10.21
	github.com/gogo/protobuf v1.3.3
	github.com/golang/protobuf v1.5.2
	github.com/gorilla/mux v1.8.0
	github.com/grpc-ecosystem/grpc-gateway v1.16.0
	github.com/prometheus/client_golang v1.14.0
	github.com/spf13/cast v1.5.0
	github.com/spf13/cobra v1.6.0
	github.com/stretchr/testify v1.8.1
	github.com/tendermint/crypto v0.0.0-20191022145703-50d29ede1e15
	github.com/tendermint/spm v0.1.9
	github.com/tendermint/tendermint v0.34.23
	github.com/tendermint/tm-db v0.6.7
	github.com/wormhole-foundation/wormhole/sdk v0.0.0-20220926172624-4b38dc650bb0
	golang.org/x/crypto v0.1.0
	golang.org/x/net v0.2.0 // indirect
	google.golang.org/genproto v0.0.0-20221114212237-e4508ebdbee1
	google.golang.org/grpc v1.50.1
	nhooyr.io/websocket v1.8.7 // indirect
)

replace (
	github.com/99designs/keyring => github.com/cosmos/keyring v1.1.7-0.20210622111912-ef00f8ac3d76
	github.com/CosmWasm/wasmd v0.30.0 => github.com/wormhole-foundation/wasmd v0.30.0-wormchain-1
	github.com/cosmos/cosmos-sdk => github.com/wormhole-foundation/cosmos-sdk v0.45.9-wormhole-2
	github.com/gogo/protobuf => github.com/regen-network/protobuf v1.3.3-alpha.regen.1
	github.com/wormhole-foundation/wormhole/sdk => ../sdk
	google.golang.org/grpc => google.golang.org/grpc v1.33.2
)
