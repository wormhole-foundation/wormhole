brew install cmake

 rustup install stable-x86_64-apple-darwin
 #rustup target add stable-x86_64-apple-darwin
 rustup target add x86_64-apple-darwin

=== Building

  % ./node_builder.sh

=== Running

  % ./start_node.sh

# If you don't remember your newly generated address

   % sui client addresses
   Showing 1 results.
   0x13b3cb89cf3226d3b860294fc75dc6c91f0c5ecf

# Give yourself some money

   % scripts/faucet.sh `sui client addresses | tail -1`

# Looking at the prefunded address

   % sui client objects --address 0x13b3cb89cf3226d3b860294fc75dc6c91f0c5ecf

=== Boot tilt

# fund our standard account

 We don't run a faucet since it doesn't always unlock the client LOCK files.  So, instead we just steal a chunk of coins
 from the default accounts created when the node was initialized.  Once sui is showing as live...

``` sh
 % kubectl exec -it sui-0 -c sui-node -- /tmp/funder.sh
```

# getting into the sui k8s node (if you need to crawl around)

   kubectl exec -it sui-0 -c sui-node -- /bin/bash
   kubectl exec -it guardian-0 -c guardiand -- /bin/bash

# setup the client.yaml

``` sh
  % rm -rf $HOME/.sui
  % sui keytool import "daughter exclude wheat pudding police weapon giggle taste space whip satoshi occur" ed25519
  % sui client
```
     point it at http://localhost:9000.  The key you create doesn't matter.

# edit $HOME/.sui/sui_config/client.yaml

``` sh
   sed -i -e 's/active_address.*/active_address: "0x13b3cb89cf3226d3b860294fc75dc6c91f0c5ecf"/' ~/.sui/sui_config/client.yaml 
```


# deploy the contract

``` sh
  % scripts/deploy.sh
```

# start the watcher

``` sh
  % . env.sh
  % python3 tests/ws.py
```

# publish a message (different window)

``` sh
  % . env.sh
  % scripts/publish_message.sh
```

==

docker run -it -v `pwd`:`pwd` -w `pwd` --net=host ghcr.io/wormhole-foundation/sui:0.16.0 bash
dnf -y install git make

``` sh
  % rm -rf $HOME/.sui
  % sui keytool import "daughter exclude wheat pudding police weapon giggle taste space whip satoshi occur" secp256k1
  % sui client
```

to get a new emitter

  kubectl exec -it sui-0 -c sui-node -- /tmp/funder.sh
  scripts/deploy.sh
  . env.sh
  sui client call --function get_new_emitter --module wormhole --package $WORM_PACKAGE --gas-budget 20000 --args \"$WORM_STATE\" 

  sui client objects
  scripts/publish_message.sh 0x165ef7366c4267c6506bcf63d2419556f34f48d6


curl -s -X POST -d '{"jsonrpc":"2.0", "id": 1, "method": "sui_getEvents", "params": [{"MoveEvent": "0xf4179152ab02e4212d7e7b20f37a9a86ab6d50fb::state::WormholeMessage"}, null, 10, true]}' -H 'Content-Type: application/json' http://127.0.0.1:9002 | jq

curl -s -X POST -d '{"jsonrpc":"2.0", "id": 1, "method": "sui_getEvents", "params": [{"Transaction": "cL+uWFEVcQrkAiOxOJmaK7JmlOJdE3/8X5JFbJwBxCQ="}, null, 10, true]}' -H 'Content-Type: application/json' http://127.0.0.1:9002 | jq

"txhash": "0x70bfae585115710ae40223b138999a2bb26694e25d137ffc5f92456c9c01c424", "txhash_b58": "8b8Bn8MUqAWeVz2BE5hMicC9KaRkV6UM4v1JLWGUjxcT", "
Digest: cL+uWFEVcQrkAiOxOJmaK7JmlOJdE3/8X5JFbJwBxCQ=

  kubectl exec -it guardian-0 -- /guardiand admin send-observation-request --socket /tmp/admin.sock 21 70bfae585115710ae40223b138999a2bb26694e25d137ffc5f92456c9c01c424

// curl -s -X POST -d '{"jsonrpc":"2.0", "id": 1, "method": "sui_getCommitteeInfo", "params": []}' -H 'Content-Type: application/json' http://127.0.0.1:9002 | jq

// curl -s -X POST -d '{"jsonrpc":"2.0", "id": 1, "method": "sui_getLatestCheckpointSequenceNumber", "params": []}' -H 'Content-Type: application/json' http://127.0.0.1:9000 
