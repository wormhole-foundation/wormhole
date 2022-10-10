# Wormhole Chain Tilt Validator Environment

The intent of this validators directory is to make a network of realish validators which will run in tiltnet, and are useable for testing purposes.

## How the Tilt environment works:

The Dockerfile in the root directory builds into the directory `/guardian_validator`. The starport build is perfectly useable in a single validator environment, but has the downside of using different tendermint keys each time as part of the starport process. This makes it tricky to write repeatable tests which deal with validator registration or bonding.

In order for our tests to run determininistically, we want to use the exact same node address, tendermint keys, operator addresses, and guardian keys every single time the tilt environment stands up. However, we also want to capture code changes to the `config.yml`, starport modules, and `cosmos-sdk` fork, so we cannot easily abandon using starport to build and initialize the chain.

To accomplish this, we first start a single validator with a fixed Tendermint address, which is specified in the `genesis.json` as both a validator and a Guardian Validator. This first validator can bootstrap the network, and allow all subsequent validators to come online and register.

Thus, `first_validator` (represented in tilt as `guardian-validator`) is a special node. Starport outputs a newly generated validator public key as the only validator in its `genesis.json`. However, the Dockerfile then explicitly runs a `collect-gentxs` command afterwards, which moves the JSON at `/first_validator/genesis/gentx.json` into the genesis file at `/guardian_validator/genesis.json`. The genesis file in the `/guardian_validator` directory is used by all the validators. Thus, after the genesis block, the `first_validator` is the only validator. All later validators come on initially with 0 voting power. The `genesis.json` (specifically, the gentxs it got from `first_validator/genesis/gentxs`) maps the `first_validator`'s tendermint key to the tiltGuardian operator address, and then the `config.yml` file maps the tiltGuardian operator address to the Tilt Guardian Public key by putting it into the genesis init of the Wormhole module.

## How to add new Validators to Tilt

1.  Generate private keys for the validator. The easiest way to do this is to add a new account to the config.yml (with a mnemonic), and then run the following command. This will generate all the private keys and config files for the node.

         make

2.  Create a new directory in the validators folder. Copy the content of the `/keyring-test` and `/config` directories inside `/build` into this newly created folder

3.  Change the `validators` target in the `Makefile` to include the new directory.

5.  Add a new kubernetes configuration for this validator into the `/kubernetes` directory.

6.  Add 87FDA636405386BF686341442ACC9FDECF9A2396@guardian-validator:26656 (the `first_validator`) to the list of `persistent_peers` in the config.toml file of the new validator. This will allow the new node to discover the other validators when it comes online.

7.  Add the new kubernetes object to the `Tiltfile`

At this point you should have a container which comes online as a non-validating node, and no additional action is needed if that is sufficient. A couple other things you may want to make note of are:

- The Tendermint ID & pubkey of the node. The IDs of the first two nodes are posted below. The easiest way to get this is to simply grab it from the logs when the validator starts up.
- Adding your validator to the genesis block. When you run `make`, it should put out a gentx file. This gentx payload needs to make its way into the `genesis.json` in order for the validator to be registered in the genesis block. The easiest way to do this would be to run the `./build/wormhole-chaind gentx` command (with the required arguments), then running `make` again.

## Validator Information

first validator:
addr=87FDA636405386BF686341442ACC9FDECF9A2396 pubKey=m9OwPF6HSFZ2sI3lUU8myhsHY2CfueG99l2IMAGgQ5g=

second validator:
addr=3C7020B8D1889869974F2A1353203D611B824525 pubKey=Zcujkt1sXRWWLfhgxLAm/Q+ioLn4wFim0OnGPLlCG0I=
