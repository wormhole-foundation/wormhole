# Wormhole CLI


## Installation

    make install

This installs two binaries, `worm-fetch-governance` and `worm` on your `$PATH`.

## Usage

### `worm-fetch-governance`

    Usage:
      worm-fetch-governance [sequence]

    Fetch a governance VAA by sequence number, and print it as hex.


For example

    worm-fetch-governance 13940208096455381020

prints

    01000000010d0012e6b39c6da90c5dfd3c228edbb78c7...


### `worm`

This is the main CLI tool. To use it, set up `$HOME/.wormhole/.env` with your
private keys, based on `.env.sample` in this folder.

    worm [command]

    Commands:
      worm generate      generate VAAs (devnet and testnet only)
      worm parse <vaa>   Parse a VAA
      worm submit <vaa>  Execute a VAA

    Options:
      --help     Show help                                  [boolean]
      --version  Show version number                        [boolean]

 Consult the `--help` flag for using subcommands.

 Use `generate` to create VAAs for testing. For example, to create an NFT bridge registration VAA:

    worm generate registration --module NFTBridge \
    --chain-id 2 \
    --contract-address 706abc4E45D419950511e474C7B9Ed348A4a716c \
    --guardian-secret cfb12303a19cde580bb4dd771639b0d26bc68353645571a8cff516ab2ee113a0

Use `parse` to parse a VAA into JSON. For example,

    worm parse $(worm-fetch-governance 13940208096455381020)
    
will fetch governance VAA `13940208096455381020` and print it as JSON.
    
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

Use `submit` to submit a VAA to a chain. It first parses the VAA and figures out
what's the destination chain and module. For example, a contract upgrade contains both the target chain and module, so the only required argument is the network moniker (`mainnet` or `testnet`):

    worm submit $(cat my-nft-registration.txt) --network mainnet


For VAAs that don't have a specific target chain (like registrations or guardian
set upgrades), the script will ask you to specify the target chain.
For example, to submit a guardian set upgrade on all chains, simply run:

    worm-fetch-governance 13940208096455381020 > guardian-upgrade.txt
    worm submit $(cat guardian-upgrade.txt) --network mainnet --chain oasis
    worm submit $(cat guardian-upgrade.txt) --network mainnet --chain aurora
    worm submit $(cat guardian-upgrade.txt) --network mainnet --chain fantom
    worm submit $(cat guardian-upgrade.txt) --network mainnet --chain karura
    worm submit $(cat guardian-upgrade.txt) --network mainnet --chain klaytn
    worm submit $(cat guardian-upgrade.txt) --network mainnet --chain avalanche
    worm submit $(cat guardian-upgrade.txt) --network mainnet --chain polygon
    worm submit $(cat guardian-upgrade.txt) --network mainnet --chain bsc
    worm submit $(cat guardian-upgrade.txt) --network mainnet --chain solana
    worm submit $(cat guardian-upgrade.txt) --network mainnet --chain terra
    worm submit $(cat guardian-upgrade.txt) --network mainnet --chain ethereum
    worm submit $(cat guardian-upgrade.txt) --network mainnet --chain celo

The VAA payload type (guardian set upgrade) specifies that this VAA should go to the core bridge, and the tool directs it there.
