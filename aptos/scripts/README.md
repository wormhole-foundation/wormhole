
Install `aptos` CLI by running
```shell
cargo install --git https://github.com/aptos-labs/aptos-core.git aptos --rev 8ba12c5badeb68d8ff4625a32aceb9043398b16b
```

Install `worm` CLI by running
```
wormhole/clients/js $ make install
```

## Development workflow

NOTE: this is in flux and likely will change often, so look back here every now
and then.

First start the local aptos validator by running

``` shell
worm start-validator aptos
```

Then build & deploy the contracts

``` shell
./deploy devnet
```

At this point you can send messages by running

``` shell
ts-node publish_wormhole_message.ts
```

### Upgrades

Make a change to the contract, then rebuild and run the upgrade script:

``` shell
./upgrade devnet Core
```

### RPC

https://fullnode.devnet.aptoslabs.com/v1/spec#/operations/get_events_by_event_handle

``` shell
curl --request GET --header 'Content-Type: application/json'  --url 'http://localhost:8080/v1/accounts/277fa055b6a73c42c0662d5236c65c864ccbf2d4abd21f174a30c8b786eab84b/events/0x277fa055b6a73c42c0662d5236c65c864ccbf2d4abd21f174a30c8b786eab84b::state::WormholeMessageHandle/event?start=0' | jq
```


