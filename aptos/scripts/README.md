
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
worm aptos start-validator
```

Then build & deploy the contracts

``` shell
./deploy devnet
```

At this point you can send messages by running

``` shell
worm aptos -n devnet send-example-message "hi mom"
```

### Upgrades

Make a change to the contract, then rebuild and run the upgrade script:

``` shell
./upgrade devnet Core
```

### RPC

https://fullnode.devnet.aptoslabs.com/v1/spec#/operations/get_events_by_event_handle

``` shell
curl --request GET --header 'Content-Type: application/json'  --url 'http://localhost:8080/v1/accounts/0xde0036a9600559e295d5f6802ef6f3f802f510366e0c23912b0655d972166017/events/0xde0036a9600559e295d5f6802ef6f3f802f510366e0c23912b0655d972166017::state::WormholeMessageHandle/event?start=0' | jq
```


