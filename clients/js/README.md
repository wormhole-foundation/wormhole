# Wormhole CLI

This tool is a command line interface to Wormhole.

## Installation

Pull down the repo if you don't already have it and cd to the appropriate directory:

    git clone https://github.com/wormhole-foundation/wormhole
    cd wormhole/clients/js

Build and install the cli tool:

    make install

This installs two binaries, `worm-fetch-governance` and `worm` on your `$PATH`.

To use `worm`, set up `$HOME/.wormhole/.env` with your
private keys, based on `.env.sample` in this folder.

## Usage

```sh
worm <command>

Commands:
  worm aptos                         Aptos utilities
  worm edit-vaa                      Edits or generates a VAA
  worm evm                           EVM utilities
  worm generate                      generate VAAs (devnet and testnet only)
  worm info                          Contract, chain, rpc and address information utilities
  worm near                          NEAR utilities
  worm parse <vaa>                   Parse a VAA (can be in either hex or base64
                                      format)
  worm recover <digest> <signature>  Recover an address from a signature
  worm submit <vaa>                  Execute a VAA
  worm sui                           Sui utilities
  worm verify-vaa                    Verifies a VAA by querying the core contract on Ethereum

Options:
  --help     Show help                                                 [boolean]
  --version  Show version number                                       [boolean]
```

### Subcommands

<!--CLI_USAGE-->
<details>
<summary> aptos </summary>

```sh
worm aptos <command>

Commands:
  worm aptos init-token-bridge              Init token bridge contract
  worm aptos init-wormhole                  Init Wormhole core contract
  worm aptos deploy <package-dir>           Deploy an Aptos package
  worm aptos deploy-resource <seed>         Deploy an Aptos package using a
  <package-dir>                             resource account
  worm aptos send-example-message           Send example message
  <message>
  worm aptos derive-resource-account        Derive resource account address
  <account> <seed>
  worm aptos derive-wrapped-address         Derive wrapped coin type
  <chain> <origin-address>
  worm aptos hash-contracts <package-dir>   Hash contract bytecodes for upgrade
  worm aptos upgrade <package-dir>          Perform upgrade after VAA has been
                                            submitted
  worm aptos migrate                        Perform migration after contract
                                            upgrade
  worm aptos faucet                         Request money from the faucet for a
                                            given account
  worm aptos start-validator                Start a local aptos validator

Options:
  --help     Show help                                                 [boolean]
  --version  Show version number                                       [boolean]
```
</details>

<details>
<summary> edit-vaa </summary>

```sh
Options:
      --help                       Show help                           [boolean]
      --version                    Show version number                 [boolean]
  -v, --vaa                        vaa in hex format         [string] [required]
  -n, --network                    Network
                            [required] [choices: "mainnet", "testnet", "devnet"]
      --guardian-set-index, --gsi  guardian set index                   [number]
      --signatures, --sigs         comma separated list of signatures   [string]
      --wormscanurl, --wsu         url to wormscan entry for the vaa that
                                   includes signatures                  [string]
      --wormscan, --ws             if specified, will query the wormscan entry
                                   for the vaa to get the signatures   [boolean]
      --emitter-chain-id, --ec     emitter chain id to be used in the vaa
                                                                        [number]
      --emitter-address, --ea      emitter address to be used in the vaa[string]
      --nonce, --no                nonce to be used in the vaa          [number]
      --sequence, --seq            sequence number to be used in the vaa[string]
      --consistency-level, --cl    consistency level to be used in the vaa
                                                                        [number]
      --timestamp, --ts            timestamp to be used in the vaa in unix
                                   seconds                              [number]
  -p, --payload                    payload in hex format                [string]
      --guardian-secret, --gs      Guardian's secret key                [string]
```
</details>

<details>
<summary> evm </summary>

```sh
worm evm <command>

Commands:
  worm evm address-from-secret <secret>  Compute a 20 byte eth address from a 32
                                         byte private key
  worm evm storage-update                Update a storage slot on an EVM fork
                                         during testing (anvil or hardhat)
  worm evm chains                        Return all EVM chains
  worm evm info                          Query info about the on-chain state of
                                         the contract
  worm evm hijack                        Override the guardian set of the core
                                         bridge contract during testing (anvil
                                         or hardhat)
  worm evm start-validator               Start a local EVM validator

Options:
  --help     Show help                                                 [boolean]
  --version  Show version number                                       [boolean]
  --rpc      RPC endpoint                                               [string]
```
</details>

<details>
<summary> generate </summary>

```sh
worm generate [command]

Commands:
  worm generate registration                Generate registration VAA
  worm generate upgrade                     Generate contract upgrade VAA
  worm generate attestation                 Generate a token attestation VAA
  worm generate recover-chain-id            Generate a recover chain ID VAA
  worm generate                             Sets the default delivery provider
  set-default-delivery-provider             for the Wormhole Relayer contract

Options:
      --help             Show help                                     [boolean]
      --version          Show version number                           [boolean]
  -g, --guardian-secret  Guardians' secret keys (CSV)        [string] [required]
```
</details>

<details>
<summary> info </summary>

```sh
worm info [command]

Commands:
  worm info chain-id <chain>                Print the wormhole chain ID integer
                                            associated with the specified chain
                                            name
  worm info contract <network> <chain>      Print contract address
  <module>
  worm info emitter <chain> <address>       Print address in emitter address
                                            format
  worm info origin <chain> <address>        Print the origin chain and address
                                            of the asset that corresponds to the
                                            given chain and address.
  worm info registrations <network>         Print chain registrations
  <chain> <module>
  worm info rpc <network> <chain>           Print RPC address
  worm info wrapped <origin-chain>          Print the wrapped address on the
  <origin-address> <target-chain>           target chain that corresponds with
                                            the specified origin chain and
                                            address.

Options:
  --help     Show help                                                 [boolean]
  --version  Show version number                                       [boolean]
```
</details>

<details>
<summary> near </summary>

```sh
worm near [command]

Commands:
  worm near contract-update <file>  Submit a contract update using our specific
                                    APIs
  worm near deploy <file>           Submit a contract update using near APIs

Options:
      --help      Show help                                            [boolean]
      --version   Show version number                                  [boolean]
  -m, --module    Module to query  [choices: "Core", "NFTBridge", "TokenBridge"]
  -n, --network   Network   [required] [choices: "mainnet", "testnet", "devnet"]
      --account   Near deployment account                    [string] [required]
      --attach    Attach some near                                      [string]
      --target    Near account to upgrade                               [string]
      --mnemonic  Near private keys                                     [string]
      --key       Near private key                                      [string]
  -r, --rpc       Override default rpc endpoint url                     [string]
```
</details>

<details>
<summary> parse <vaa> </summary>

```sh
Positionals:
  vaa  vaa                                                              [string]

Options:
  --help     Show help                                                 [boolean]
  --version  Show version number                                       [boolean]
```
</details>

<details>
<summary> recover <digest> <signature> </summary>

```sh
Positionals:
  digest     digest                                                     [string]
  signature  signature                                                  [string]

Options:
  --help     Show help                                                 [boolean]
  --version  Show version number                                       [boolean]
```
</details>

<details>
<summary> submit <vaa> </summary>

```sh
Positionals:
  vaa  vaa                                                              [string]

Options:
      --help              Show help                                    [boolean]
      --version           Show version number                          [boolean]
  -c, --chain             chain name
             [choices: "unset", "solana", "ethereum", "terra", "bsc", "polygon",
        "avalanche", "oasis", "algorand", "aurora", "fantom", "karura", "acala",
            "klaytn", "celo", "near", "moonbeam", "neon", "terra2", "injective",
         "osmosis", "sui", "aptos", "arbitrum", "optimism", "gnosis", "pythnet",
         "xpla", "btc", "base", "sei", "rootstock", "scroll", "mantle", "blast",
    "xlayer", "linea", "berachain", "seievm", "wormchain", "cosmoshub", "evmos",
               "kujira", "neutron", "celestia", "stargaze", "seda", "dymension",
                    "provenance", "sepolia", "arbitrum_sepolia", "base_sepolia",
                               "optimism_sepolia", "holesky", "polygon_sepolia"]
  -n, --network           Network
                            [required] [choices: "mainnet", "testnet", "devnet"]
  -a, --contract-address  Contract to submit VAA to (override config)   [string]
      --rpc               RPC endpoint                                  [string]
      --all-chains, --ac  Submit the VAA to all chains except for the origin
                          chain specified in the payload
                                                      [boolean] [default: false]
```
</details>

<details>
<summary> sui </summary>

```sh
worm sui <command>

Commands:
  worm sui build-coin                    Build wrapped coin and dump bytecode.

                                         Example:
                                         worm sui build-coin -d 8 -v V__0_1_1 -n
                                         testnet -r
                                         "https://fullnode.testnet.sui.io:443"
  worm sui deploy <package-dir>          Deploy a Sui package
  worm sui init-example-message-app      Initialize example core message app
  worm sui init-token-bridge             Initialize token bridge contract
  worm sui init-wormhole                 Initialize wormhole core contract
  worm sui publish-example-message       Publish message from example app via
                                         core bridge
  worm sui setup-devnet                  Setup devnet by deploying and
                                         initializing core and token bridges and
                                         submitting chain registrations.
  worm sui objects <owner>               Get owned objects by owner
  worm sui package-id <state-object-id>  Get package ID from State object ID
  worm sui tx <transaction-digest>       Get transaction details

Options:
  --help     Show help                                                 [boolean]
  --version  Show version number                                       [boolean]
```
</details>

<details>
<summary> transfer </summary>

```sh
Options:
      --help        Show help                                          [boolean]
      --version     Show version number                                [boolean]
      --src-chain   source chain
           [required] [choices: "solana", "ethereum", "terra", "bsc", "polygon",
        "avalanche", "oasis", "algorand", "aurora", "fantom", "karura", "acala",
            "klaytn", "celo", "near", "moonbeam", "neon", "terra2", "injective",
         "osmosis", "sui", "aptos", "arbitrum", "optimism", "gnosis", "pythnet",
         "xpla", "btc", "base", "sei", "rootstock", "scroll", "mantle", "blast",
    "xlayer", "linea", "berachain", "seievm", "wormchain", "cosmoshub", "evmos",
               "kujira", "neutron", "celestia", "stargaze", "seda", "dymension",
                    "provenance", "sepolia", "arbitrum_sepolia", "base_sepolia",
                               "optimism_sepolia", "holesky", "polygon_sepolia"]
      --dst-chain   destination chain
           [required] [choices: "solana", "ethereum", "terra", "bsc", "polygon",
        "avalanche", "oasis", "algorand", "aurora", "fantom", "karura", "acala",
            "klaytn", "celo", "near", "moonbeam", "neon", "terra2", "injective",
         "osmosis", "sui", "aptos", "arbitrum", "optimism", "gnosis", "pythnet",
         "xpla", "btc", "base", "sei", "rootstock", "scroll", "mantle", "blast",
    "xlayer", "linea", "berachain", "seievm", "wormchain", "cosmoshub", "evmos",
               "kujira", "neutron", "celestia", "stargaze", "seda", "dymension",
                    "provenance", "sepolia", "arbitrum_sepolia", "base_sepolia",
                               "optimism_sepolia", "holesky", "polygon_sepolia"]
      --dst-addr    destination address                      [string] [required]
      --token-addr  token address               [string] [default: native token]
      --amount      token amount                             [string] [required]
  -n, --network     Network [required] [choices: "mainnet", "testnet", "devnet"]
      --rpc         RPC endpoint                                        [string]
```
</details>

<details>
<summary> verify-vaa </summary>

```sh
Options:
      --help     Show help                                             [boolean]
      --version  Show version number                                   [boolean]
  -v, --vaa      vaa in hex format                           [string] [required]
  -n, --network  Network    [required] [choices: "mainnet", "testnet", "devnet"]
```
</details>

<details>
<summary> status <network> <chain> <tx> </summary>

```sh
Positionals:
  network  Network                     [choices: "mainnet", "testnet", "devnet"]
  chain    Source chain
             [choices: "unset", "solana", "ethereum", "terra", "bsc", "polygon",
        "avalanche", "oasis", "algorand", "aurora", "fantom", "karura", "acala",
            "klaytn", "celo", "near", "moonbeam", "neon", "terra2", "injective",
         "osmosis", "sui", "aptos", "arbitrum", "optimism", "gnosis", "pythnet",
         "xpla", "btc", "base", "sei", "rootstock", "scroll", "mantle", "blast",
    "xlayer", "linea", "berachain", "seievm", "wormchain", "cosmoshub", "evmos",
               "kujira", "neutron", "celestia", "stargaze", "seda", "dymension",
                    "provenance", "sepolia", "arbitrum_sepolia", "base_sepolia",
                               "optimism_sepolia", "holesky", "polygon_sepolia"]
  tx       Source transaction hash                                      [string]

Options:
  --help     Show help                                                 [boolean]
  --version  Show version number                                       [boolean]
```
</details>
<!--CLI_USAGE-->

## Examples

### VAA generation

Use `generate` to create VAAs for testing. For example, to create an NFT bridge registration VAA:

```sh
$ worm generate registration --module NFTBridge \
    --chain bsc \
    --contract-address 0x706abc4E45D419950511e474C7B9Ed348A4a716c \
    --guardian-secret cfb12303a19cde580bb4dd771639b0d26bc68353645571a8cff516ab2ee113a0
```

Example creating a token attestation VAA:

```sh
$ worm generate attestation --emitter-chain ethereum \
    --emitter-address 11111111111111111111111111111115 \
    --chain ethereum \
    --token-address 0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48 \
    --decimals 6 \
    --symbol USDC \
    --name USDC \
    --guardian-secret cfb12303a19cde580bb4dd771639b0d26bc68353645571a8cff516ab2ee113a0
```

### VAA parsing

Use `parse` to parse a VAA into JSON. For example,

    worm parse $(worm-fetch-governance 13940208096455381020)

will fetch governance VAA `13940208096455381020` and print it as JSON.

```sh
# ...signatures elided
timestamp: 1651416474,
nonce: 1570649151,
emitterChain: 1,
emitterAddress: '0000000000000000000000000000000000000000000000000000000000000004',
sequence: 13940208096455381020n,
consistencyLevel: 32,
payload: {
  module: 'Core',
  type: 'GuardianSetUpgrade',
  chain: 0,
  newGuardianSetIndex: 2,
  newGuardianSetLength: 19,
  newGuardianSet: [
    '58cc3ae5c097b213ce3c81979e1b9f9570746aa5',
    'ff6cb952589bde862c25ef4392132fb9d4a42157',
    '114de8460193bdf3a2fcf81f86a09765f4762fd1',
    '107a0086b32d7a0977926a205131d8731d39cbeb',
    '8c82b2fd82faed2711d59af0f2499d16e726f6b2',
    '11b39756c042441be6d8650b69b54ebe715e2343',
    '54ce5b4d348fb74b958e8966e2ec3dbd4958a7cd',
    '66b9590e1c41e0b226937bf9217d1d67fd4e91f5',
    '74a3bf913953d695260d88bc1aa25a4eee363ef0',
    '000ac0076727b35fbea2dac28fee5ccb0fea768e',
    'af45ced136b9d9e24903464ae889f5c8a723fc14',
    'f93124b7c738843cbb89e864c862c38cddcccf95',
    'd2cc37a4dc036a8d232b48f62cdd4731412f4890',
    'da798f6896a3331f64b48c12d1d57fd9cbe70811',
    '71aa1be1d36cafe3867910f99c09e347899c19c3',
    '8192b6e7387ccd768277c17dab1b7a5027c0b3cf',
    '178e21ad2e77ae06711549cfbb1f9c7a9d8096e8',
    '5e1487f35515d02a92753504a8d75471b9f49edb',
    '6fbebc898f403e4773e95feb15e80c9a99c8348d'
  ]
}
```

### Submitting VAAs

Use `submit` to submit a VAA to a chain. It first parses the VAA and figures out
what's the destination chain and module. For example, a contract upgrade contains both the target chain and module, so the only required argument is the network moniker (`mainnet` or `testnet`):

    worm submit $(cat my-nft-registration.txt) --network mainnet

For VAAs that don't have a specific target chain (like registrations or guardian
set upgrades), the script will ask you to specify the target chain.
For example, to submit a guardian set upgrade on all chains, simply run:

```sh
$ worm-fetch-governance 13940208096455381020 > guardian-upgrade.txt
$ worm submit $(cat guardian-upgrade.txt) --network mainnet --chain oasis
$ worm submit $(cat guardian-upgrade.txt) --network mainnet --chain aurora
$ worm submit $(cat guardian-upgrade.txt) --network mainnet --chain fantom
$ worm submit $(cat guardian-upgrade.txt) --network mainnet --chain karura
$ worm submit $(cat guardian-upgrade.txt) --network mainnet --chain acala
$ worm submit $(cat guardian-upgrade.txt) --network mainnet --chain klaytn
$ worm submit $(cat guardian-upgrade.txt) --network mainnet --chain avalanche
$ worm submit $(cat guardian-upgrade.txt) --network mainnet --chain polygon
$ worm submit $(cat guardian-upgrade.txt) --network mainnet --chain bsc
$ worm submit $(cat guardian-upgrade.txt) --network mainnet --chain solana
$ worm submit $(cat guardian-upgrade.txt) --network mainnet --chain terra
$ worm submit $(cat guardian-upgrade.txt) --network mainnet --chain ethereum
$ worm submit $(cat guardian-upgrade.txt) --network mainnet --chain celo
```

The VAA payload type (guardian set upgrade) specifies that this VAA should go to the core bridge, and the tool directs it there.

### info

To get info about a contract (only EVM supported at this time)

```sh
$ worm evm info -c bsc -n mainnet -m TokenBridge

{
  "address": "0xB6F6D86a8f9879A9c87f643768d9efc38c1Da6E7",
  "wormhole": "0x98f3c9e6E3fAce36bAAd05FE09d375Ef1464288B",
  "implementation": "0xEe91C335eab126dF5fDB3797EA9d6aD93aeC9722",
  "isInitialized": true,
  "tokenImplementation": "0xb6D7bbdE7c46a8B784F4a19C7FDA0De34b9577DB",
  "chainId": 4,
  "governanceChainId": 1,
  "governanceContract": "0x0000000000000000000000000000000000000000000000000000000000000004",
  "WETH": "0xbb4CdB9CBd36B01bD1cBaEBF2De08d9173bc095c",
  "registrations": {
    "solana": "0xec7372995d5cc8732397fb0ad35c0121e0eaa90d26f828a534cab54391b3a4f5",
    "ethereum": "0x0000000000000000000000003ee18b2214aff97000d974cf647e7c347e8fa585",
    "terra": "0x0000000000000000000000007cf7b764e38a0a5e967972c1df77d432510564e2",
    "polygon": "0x0000000000000000000000005a58505a96d1dbf8df91cb21b54419fc36e93fde",
    "avalanche": "0x0000000000000000000000000e082f06ff657d94310cb8ce8b0d9a04541d8052",
    "oasis": "0x0000000000000000000000005848c791e09901b40a9ef749f2a6735b418d7564",
    "algorand": "0x0000000000000000000000000000000000000000000000000000000000000000",
    "aurora": "0x00000000000000000000000051b5123a7b0f9b2ba265f9c4c8de7d78d52f510f",
    "fantom": "0x0000000000000000000000007c9fc5741288cdfdd83ceb07f3ea7e22618d79d2",
    "karura": "0x000000000000000000000000ae9d7fe007b3327aa64a32824aaac52c42a6e624",
    "acala": "0x000000000000000000000000ae9d7fe007b3327aa64a32824aaac52c42a6e624",
    "klaytn": "0x0000000000000000000000005b08ac39eaed75c0439fc750d9fe7e1f9dd0193f",
    "celo": "0x000000000000000000000000796dff6d74f3e27060b71255fe517bfb23c93eed",
    "near": "0x0000000000000000000000000000000000000000000000000000000000000000",
    "injective": "0x0000000000000000000000000000000000000000000000000000000000000000",
    "osmosis": "0x0000000000000000000000000000000000000000000000000000000000000000",
    "sui": "0x0000000000000000000000000000000000000000000000000000000000000000",
    "aptos": "0x0000000000000000000000000000000000000000000000000000000000000000",
    "moonbeam": "0x0000000000000000000000000000000000000000000000000000000000000000",
    "neon": "0x0000000000000000000000000000000000000000000000000000000000000000",
    "terra2": "0xa463ad028fb79679cfc8ce1efba35ac0e77b35080a1abe9bebe83461f176b0a3",
    "arbitrum": "0x0000000000000000000000000000000000000000000000000000000000000000",
    "optimism": "0x0000000000000000000000000000000000000000000000000000000000000000",
    "gnosis": "0x0000000000000000000000000000000000000000000000000000000000000000",
  }
}

```

### Misc

To get the contract address for a module:

    $ worm info contract mainnet bsc NFTBridge

To get the RPC address for a chain

    $ worm info rpc mainnet bsc
