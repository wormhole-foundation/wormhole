module github.com/certusone/wormhole/clients/eth

go 1.20

require (
	github.com/certusone/wormhole/node v0.0.0-20210722131135-a191017d22d0
	github.com/ethereum/go-ethereum v1.12.0
	github.com/spf13/cobra v1.7.0
	github.com/wormhole-foundation/wormhole/sdk v0.0.0-20220926172624-4b38dc650bb0
)

replace github.com/certusone/wormhole/node => ../../node

// See https://github.com/cosmos/cosmos-sdk/issues/10925 for more details.
// Won't be needed anymore as soon as github.com/certusone/wormhole/node upgrades to v0.47
replace github.com/gogo/protobuf => github.com/regen-network/protobuf v1.3.3-alpha.regen.1

replace github.com/wormhole-foundation/wormhole/sdk => ../../sdk

require (
	github.com/StackExchange/wmi v1.2.1 // indirect
	github.com/btcsuite/btcd/btcec/v2 v2.3.2 // indirect
	github.com/deckarep/golang-set/v2 v2.1.0 // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.1.0 // indirect
	github.com/fsnotify/fsnotify v1.6.0 // indirect
	github.com/go-ole/go-ole v1.2.6 // indirect
	github.com/go-stack/stack v1.8.1 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/gorilla/websocket v1.5.0 // indirect
	github.com/holiman/uint256 v1.2.2-0.20230321075855-87b91420868c // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/shirou/gopsutil v3.21.4-0.20210419000835-c7a38de76ee5+incompatible // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/tklauser/go-sysconf v0.3.10 // indirect
	github.com/tklauser/numcpus v0.4.0 // indirect
	golang.org/x/crypto v0.7.0 // indirect
	golang.org/x/sys v0.7.0 // indirect
	golang.org/x/text v0.9.0 // indirect
	gopkg.in/natefinch/npipe.v2 v2.0.0-20160621034901-c1b8fa8bdcce // indirect
)
