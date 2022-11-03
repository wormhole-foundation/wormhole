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
   0x2acab6bb0e4722e528291bc6ca4f097e18ce9331

# Give yourself some money

   % scripts/faucet.sh `sui client addresses | tail -1`

# Looking at the prefunded address

   % sui client objects --address 0x2acab6bb0e4722e528291bc6ca4f097e18ce9331

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
  % sui keytool import "daughter exclude wheat pudding police weapon giggle taste space whip satoshi occur" secp256k1
  % sui client
```
     point it at http://localhost:9002.  The key you create doesn't matter.

# edit $HOME/.sui/sui_config/client.yaml

``` sh
    active_address: "0x2acab6bb0e4722e528291bc6ca4f097e18ce9331"
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
