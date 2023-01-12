# Installation
Make sure your Cargo version is at least 1.64.0 and then follow the steps below:
- https://docs.sui.io/build/install


# Sui CLI
- do `sui start` to spin up a local network
- do `rpc-server` to start a server for handling rpc calls
- do `sui-faucet` to start a faucet for requesting funds from active-address

# TODOs
- The move dependencies are currently pinned to a version that matches the
  docker image for reproducibility. These should be regularly updated to track
  any upstream changes before the mainnet release.
