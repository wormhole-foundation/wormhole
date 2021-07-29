module github.com/certusone/wormhole/bridge

go 1.16

require (
	github.com/cenkalti/backoff/v4 v4.1.1
	github.com/davecgh/go-spew v1.1.1
	github.com/ethereum/go-ethereum v1.10.6
	github.com/gagliardetto/solana-go v0.3.5-0.20210727215348-0cf016734976
	github.com/gorilla/mux v1.7.4
	github.com/gorilla/websocket v1.4.2
	github.com/ipfs/go-log/v2 v2.3.0
	github.com/libp2p/go-libp2p v0.14.4
	github.com/libp2p/go-libp2p-connmgr v0.2.4
	github.com/libp2p/go-libp2p-core v0.8.6
	github.com/libp2p/go-libp2p-kad-dht v0.12.2
	github.com/libp2p/go-libp2p-pubsub v0.5.0
	github.com/libp2p/go-libp2p-quic-transport v0.11.2
	github.com/libp2p/go-libp2p-tls v0.1.3
	github.com/miguelmota/go-ethereum-hdwallet v0.1.0
	github.com/mitchellh/go-homedir v1.1.0
	github.com/mr-tron/base58 v1.2.0
	github.com/multiformats/go-multiaddr v0.3.3
	github.com/near/borsh-go v0.3.0
	github.com/prometheus/client_golang v1.10.0
	github.com/spf13/cobra v1.1.1
	github.com/spf13/viper v1.7.1
	github.com/status-im/keycard-go v0.0.0-20200402102358-957c09536969
	github.com/stretchr/testify v1.7.0
	github.com/terra-project/terra.go v1.0.1-0.20210129055710-7a586e5e027a
	github.com/tidwall/gjson v1.8.1
	go.uber.org/zap v1.16.0
	golang.org/x/crypto v0.0.0-20210513164829-c07d793c2f9a
	golang.org/x/sys v0.0.0-20210514084401-e8d321eab015
	google.golang.org/genproto v0.0.0-20200526211855-cb27e3aa2013
	google.golang.org/grpc v1.33.2
	google.golang.org/protobuf v1.26.0
)

// Temporary fork that adds GetConfirmedTransactionWithOpts. Can be removed
// once Solana mainnet has upgraded to v1.7.x.
replace github.com/gagliardetto/solana-go => github.com/certusone/solana-go v0.3.7-0.20210729105530-67b495e4e529
