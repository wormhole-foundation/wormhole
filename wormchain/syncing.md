# Syncing Mainnet Wormchain

# Contents
* [Sync From Snapshot](#sync-from-snapshot)
* [Sync Manually](#sync-manually)

# Sync From Snapshot

## Build Latest Wormchain Release

```bash
# checkout git repository
git clone https://github.com/wormhole-foundation/wormhole

# checkout latest release (v2.23.0 at time of writing)
cd wormhole
git checkout v2.23.0

# build wormchain
cd wormchain
make build/wormchaind
```

## Get a Recent Wormchain Snapshot

Please ask the Wormhole contributors for a recent Wormchain snapshot to use.

You can also use daily snapshots exported by the CryptoCrew team here: [https://github.com/clemensgg/CryptoCrew-Validators/blob/main/chains/wormchain/service_Node_Snapshot.md](https://github.com/clemensgg/CryptoCrew-Validators/blob/main/chains/wormchain/service_Node_Snapshot.md)

## Download and Clean Snapshot

After you download the snapshot, you'll need to clear the wasm cache:

```bash
# before starting your node
rm -r /data/wasm/cache
```

Now you should be able to successfully start your node!

### Troubleshooting

> We were just able to successfully statesync when retaining the `wasm` folder and deleting the wasm cache, just fyi:
> 
> - you need the `wasm` folder from a synced node
> - then configure statesync + `unsafe-reset-all`
> - then copy in wasm state `cp -r <your-wasm-folder-location> $DAEMON_HOME/data`
> - delete wasm cache `rm -r $DAEMON_HOME/data/wasm/cache`
> - then start the node & statesync

# Sync Manually

## Build Wormchain Versions

```bash
cd wormhole/wormchain

# v2.14.7
git checkout v2.14.7
make build/wormchaind
mv build/wormchaind build/wormchaind-v2.14.7

# v2.14.9.1
git checkout v2.14.9.1
make build/wormchaind
mv build/wormchaind build/wormchaind-v2.14.9.1

# v2.14.9.6
git checkout v2.14.9.6
make build/wormchaind
mv build/wormchaind build/wormchaind-v2.14.9.6

# v2.18.1
git checkout v2.18.1
make build/wormchaind
mv build/wormchaind build/wormchaind-v2.18.1

# v2.23.0
git checkout v2.23.0
make build/wormchaind
mv build/wormchaind build/wormchaind-v2.23.0
```

## Setup Folders to sync Wormchain

```bash
cd wormhole/wormchain/build
rm -rf config/
rm -rf data/
rm -rf keyring-test/
./wormchaind-v2.14.7 init node-client --chain-id wormchain --home .
rm config/config.toml
rm config/genesis.json
cp ../mainnet/* config/
```

## Sync with v2.14.7

```bash
./wormchaind-v2.14.7 start --home . --moniker <your-moniker>
# check sync status
./wormchaind-v2.14.7 status | jq ".SyncInfo.latest_block_height"
# Error in validation err="wrong Block.Header.LastResultsHash.  Expected 2AC3E9F6684C828DDBF5A990EE582FD1968DF9158845986AE01889AFDFE0CF8D, got 8161A8789F1A9404B445CDBE7EC97FC8230E89C49C649292E6A771179448D7B0" module=blockchain
# Block 1672360
```

## Sync with v2.14.9.1

```bash
./wormchaind-v2.14.9.1 rollback --home .
./wormchaind-v2.14.9.1 start --home . --moniker <your-moniker>
# check sync status
./wormchaind-v2.14.9.1 status | jq ".SyncInfo.latest_block_height"
# Error in validation err="wrong Block.Header.LastResultsHash.  Expected 9F7AC20D2E6D4D06C8A55F7F7F1CDFDC194E3F7E6F89FD9FAF73CBA35D52CDF8, got A05378138E95B21D7D04A5688BAB3578DDC8424EAD5EA1DA55F8B8C5FEE3450C" module=blockchain
# Block 2157092
```

## Sync with v2.14.9.6

```bash
./wormchaind-v2.14.9.6 rollback --home .
./wormchaind-v2.14.9.6 start --home . --moniker <your-moniker>
# sync until block 3151200 (problematic block is 3151174)
# stop the sync right after 3151174
```

## Sync with v2.18.1

```bash
./build/wormchaind-v2.18.1 start --home . --moniker <your-moniker>
# sync will stop automatically on block 4449129
# after this stops, you just need to restart your wormchain node with v2.23.0
# no rollback is required.
```

## Sync with v2.23.0

```bash
./build/wormchaind-v2.23.0 start --home . --moniker <your-moniker>
# this will sync until the latest block!
```