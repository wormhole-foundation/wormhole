brew install cmake

 rustup install stable-x86_64-apple-darwin
 #rustup target add stable-x86_64-apple-darwin
 rustup target add x86_64-apple-darwin

=== Building

  % ./node_builder.sh

=== Running

  % ./start_node.sh

=== Running under tilt

# getting into the sui k8s node (if you need to crawl around)

   kubectl exec -it sui-0 -c sui-node -- /bin/bash

# Clean up the local sui_config from previous runs

   % rm -rf ~/.sui/sui_config

# Set up your client config

   % sui client
   Config file ["/home/jsiegel/.sui/sui_config/client.yaml"] doesn't exist, do you want to connect to a Sui RPC server [yN]?y
   Sui RPC server Url (Default to Sui DevNet if not specified) : http://localhost:5001
   Select key scheme to generate keypair (0 for ed25519, 1 for secp256k1):
   1
   Generated new keypair for address with scheme "secp256k1" [0x2acab6bb0e4722e528291bc6ca4f097e18ce9331]
   Secret Recovery Phrase : [...]

# If you don't remember your newly generated address

   % sui client addresses
   Showing 1 results.
   0x2acab6bb0e4722e528291bc6ca4f097e18ce9331

# Importing prefunded address

   % sui keytool import "daughter exclude wheat pudding police weapon giggle taste space whip satoshi occur" secp256k1

# Looking at the prefunded address

   % sui client objects --address 0x2acab6bb0e4722e528291bc6ca4f097e18ce9331

# Give yourself some money

   % scripts/faucet.sh 0x2acab6bb0e4722e528291bc6ca4f097e18ce9331   

# Deploy wormhole

   % scripts/deploy.sh

# Publishing a message

   % scripts/publish_message.sh


