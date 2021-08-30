module github.com/certusone/wormhole/clients/eth

go 1.17

require (
	github.com/certusone/wormhole/node v0.0.0-20210722131135-a191017d22d0
	github.com/ethereum/go-ethereum v1.10.6
	github.com/spf13/cobra v1.1.1
)

replace github.com/certusone/wormhole/node => ../../node
