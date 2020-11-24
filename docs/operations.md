# Running a Wormhole node

![](images/nodearchitecture.svg)

## Connected chains

In addition to Wormhole itself, you need to run your own verifying node for every chain that Wormhole connects to:

- **Solana**. There is no light client for Solana yet, so you'll have to run a full solana-validator node. It does not
  have to actually be a validator - you can run solana-validator in non-validating mode if you are not a validator.

  Refer to the [Solana documentation](https://docs.solana.com/running-validator) on how to run a validator. The validator
  requirements as stated in their docs are excessive - for the current iteration for mainnet-beta, the "low end" config
  with no GPU is perfectly adequate, and will have enough spare capacity.
  [Solana's Discord server](https://solana.com/community) is a great resource for questions regarding validator ops.

- **Ethereum**. See below - you need at least a light client. For stability reasons, a full node is recommended.

- \[**Terra** requires a full node and an [LCD server](https://docs.terra.money/terracli/lcd.html#light-client-daemon)
  pointing to your full node. Refer to the [Terra documentation](https://docs.terra.money/node/join-network.html)
  on how to run a full node. From a security point of view, running only an LCD server with `--trust-node=false` pointed
  to somebody else's full node would be sufficient, but you'd then depend on that single node for availability unless
  you set up a load balancer pointing to a set of nodes.\]

Do NOT use third-party RPC service providers for any of the chains! You'd fully trust them and they could lie to you on
whether a lockup has actually been observed, and the whole point of Wormhole is to not rely on centralized nodes.

### Ethereum node requirements

In order to observe events on the Ethereum chain, you need access to an Ethereum RPC endpoint. We use geth, but for the
sake of diversity, you may want to run something that isn't geth.

With RPC providers such as Alchemy, Infura, etc. you trust those operators to provide you with untampered chain data and
have no way of verifying correctness. Therefore, Wormhole requires either an Ethereum full-node or a light-client. The
node can be operated in the full, quick or light modes with no impact on security or performance of the bridge software.
As long as the node supports the Ethereum JSON RPC API, it will be compatible with the bridge so all major
implementations will work fine.

Generally, full-nodes will work better and be more reliable than light clients which are susceptible to DoS attacks 
since only very few nodes support the light client protocol.

Running a full node typically requires ~500G of SSD storage, 8G of RAM and 4-8 CPU threads (depending on clock
frequency). Light clients have much lower hardware requirements.

## Building

For security reasons, we do not provide pre-built binaries. You need to check out the repo and build the
Wormhole binaries from source. A Git repo is much harder to tamper with than release binaries.

To build Wormhole, you need:

- [Go](https://golang.org/dl/) >= 1.15.3
- [Rust](https://www.rust-lang.org/learn/get-started) >= 1.47.0

If your Linux distribution has recent enough packages for these, it's preferable to use those and avoid
the extra third-party build dependency.

First, check out the version of the Wormhole repo that you want to deploy:

```bash
git clone https://github.com/certusone/wormhole && cd wormhole
git checkout <the stable tag you want to compile>
```

Then, compile the release binaries as an unprivileged build user:

```bash
# this doesn't actually work yet - the Makefile has yet to be written
# TOOD: https://github.com/certusone/wormhole/issues/120
make agent bridge

# Install binaries to /usr/local/bin in case of a local compilation workflow.
sudo make install
```
    
You'll end up with the following binaries in `build/`:

- `wormhole-guardiand` is the main Wormhole bridge node software.
- `wormhole-solana-agent` is a helper service which runs alongside Wormhole and exposes a gRPC API
  for Wormhole to interact with Solana and the Wormhole contract on Solana.
- `wormhole-solana-cli` is a CLI you can use to manually interact with the Wormhole contract on Solana.
  You don't strictly need this on a production machine, but it can be useful for debugging.
  
Consider these recommendations, not a tutorial to be followed blindly. You'll want to integrate this with your
existing build pipeline. If you need Dockerfile examples, you can take a look at our devnet deployment.

## Key Generation

To generate a guardian key, you only need to only build the Go bridge.

It requires [Go](https://golang.org/dl/) >= 1.15.3. Clone the Wormhole repo
and build the binary:

    git clone https://github.com/certusone/wormhole
    cd wormhole/
    ./generate-protos.sh
    cd bridge/
    go build github.com/certusone/wormhole/bridge

Then, generate a new key:

    ./bridge keygen --desc "Testnet key foo" /path/to/your.key

The key file includes a human-readable part that includes the public key
and the description.

## Deploying

⚠ TODO:️ _systemd service file examples (not entirely trivial)_  

### Kubernetes

Refer to [devnet/](../devnet) for example k8s deployments as a starting point for your own production deployment. You'll
have to build your own containers. Unless you already run Kubernetes in production, we strongly recommend a traditional
deployment on a dedicated instance - it's easier to understand and troubleshoot.

## Key Management

You'll have to manage the following keys:

 - The **guardian key**, which is the bridge consensus key. This key is very critical - your node uses it to certify
   VAA messages. The public key is stored in the guardian set on all connected chains. It does not accrue rewards.
   It's your share of the multisig mechanism that protect the Wormhole network. The guardian set can be replaced
   if a majority of the guardians agree to sign and publish a new guardian set.
  
 - A **node key**, which identifies it on the gossip network, similar to Solana's node identity or a Tendermint
   node key. It is used by the peer-to-peer network for routing and transport layer encryption.
   An attacker could potentially use it to censor your messages on the network. Other than that, it's not very
   critical and can be rotated. The node will automatically create a node key at the path you specify if it doesn't exist.
 
 - The **Solana fee payer** account supplied to wormhole-solana-agent. This is a hot wallet which should hold
   ~10 SOL to pay for VAA submissions. The Wormhole protocol includes a subsidization mechanism which uses transfer
   fees to reimburse guardians, so during normal operation, you shouldn't have to top up the account (but by
   all means, set up monitoring for it!).
   
 - _\[The **Terra fee payer** account. Terra support is still a work in progress - more details on this later\]._ 

For production, we strongly recommend to either encrypt your disks, and/or take care to never have keys touch the disk.
One way to accomplish is to store keys on an in-memory ramfs, which can't be swapped out, and restore it from cold
storage or an HSM/vault whenever the node is rebooted. You might want to disable swap altogether. None of that is
specific to Wormhole - this applies to any hot keys.

Our node software takes extra care to lock memory using mlock(2) to prevent keys from being swapped out to disk, which
is why it requires extra capabilities.

Storing keys on an HSM or using remote signers only partially mitigates the risk of server compromise - it means the key
can't get stolen, but an attacker could still cause the HSM to sign malicious payloads. Future iterations of Wormhole
may include support for remote signing using a signer like [SignOS](https://certus.one/sign-os/).

## High Availability
 
Multiple nodes with different node keys can share the same guardian keys. The node which first submits a signature
"wins" and the duplicate signature will be ignored by the network. Wormhole has no need for slashing, and therefore,
there's no risk of equivocation.

⚠️ _This is not yet tested - see https://github.com/certusone/wormhole/issues/73_  

## Monitoring

The node exposes a Prometheus endpoint for monitoring.

⚠ TODO:️ _Actually build and document this: https://github.com/certusone/wormhole/issues/11_
