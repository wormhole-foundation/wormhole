
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
   "xpla", "btc", "base", "sei", "rootstock", "wormchain", "cosmoshub", "evmos",
                                                            "kujira", "sepolia"]
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
   "xpla", "btc", "base", "sei", "rootstock", "wormchain", "cosmoshub", "evmos",
                                                            "kujira", "sepolia"]
      --dst-chain   destination chain
           [required] [choices: "solana", "ethereum", "terra", "bsc", "polygon",
        "avalanche", "oasis", "algorand", "aurora", "fantom", "karura", "acala",
            "klaytn", "celo", "near", "moonbeam", "neon", "terra2", "injective",
         "osmosis", "sui", "aptos", "arbitrum", "optimism", "gnosis", "pythnet",
   "xpla", "btc", "base", "sei", "rootstock", "wormchain", "cosmoshub", "evmos",
                                                            "kujira", "sepolia"]
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
   "xpla", "btc", "base", "sei", "rootstock", "wormchain", "cosmoshub", "evmos",
                                                            "kujira", "sepolia"]
  tx       Source transaction hash                                      [string]

Options:
  --help     Show help                                                 [boolean]
  --version  Show version number                                       [boolean]
```
</details>
