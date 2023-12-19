# Token contract verification on Etherscan

This document explains how to perform verification of the wrapped asset
contracts deployed by the Portal token bridge on Etherscan and similar explorer
services. The purpose of contract verification is to provide the source code of
the contract so that users have better transparency into the programs they
interact with (whether this makes a difference in practice is beyond the scope
of this document).

Verification requires uploading the source code and supplying the compiler
version and arguments that the deployed bytecode was compiled with. Etherscan
will then recompile the source code on their servers and match the result
against the deployed bytecode.

We will use the flattened source file, as it's easier to upload. Flattened just
means that the whole contract, including its dependencies, are inlined into a
single source file.
Wormhole's flattened source files can be downloaded from [Github
Actions](https://github.com/wormhole-foundation/wormhole/actions/workflows/evm-flattened-contracts.yml).
This page includes a version of the contracts for each ethereum contract
deployment. The deployed token contract tends not to change very often (if at
all), so it should be safe to just pick the latest run and download the
artifacts from there (download `flattened-contracts` at the bottom of the page
after selecting the run).

The file needed for token verification is `token/Token.sol`.

For our running example, we'll use the wrapped WETH contract deployed on
Moonbeam:
https://moonscan.io/address/0xab3f0245b83feb11d15aaffefd7ad465a59817ed

First, we verify that this contract is a proxy. This step can be done by
visiting the https://moonbeam.moonscan.io/proxycontractchecker page, and pasting
in the contract address. For ethereum, the site to use would be
https://etherscan.io/proxycontractchecker, and other chain explorers have
similar pages too.

This will present a popup with the following text:

> The proxy contract verification completed with the message:
> The proxy's (0xab3f0245b83feb11d15aaffefd7ad465a59817ed) implementation contract is found at: 0xb1731c586ca89a23809861c6103f0b96b3f57d92

Click save.

Next, we'll verify the actual source code. Head over to
https://moonscan.io/verifyContract and paste in contract address, in our case
`0xab3f0245b83feb11d15aaffefd7ad465a59817ed`. Fill in the rest of the form with
the following values, then continue.

| Field            | Value                     |
| ---------------- | ------------------------- |
| Compiler type    | Solidity (Single file)    |
| Compiler version | v0.8.4+commit.c7e474f2    |
| License type     | Apache-2                  |

On the next page, select "optimizations: yes", and paste the contents of
`token/Token.sol` from before into the source file textarea.

In the misc settings, enter 200 for optimization runs, and leave the rest as
default.

## ABI-encoded constructor arguments

The last missing piece is the ABI-encoded constructor arguments field.  These
are the arguments that the contract was instantiated with, and it will be
different for each wrapped contract.
There are two ways to proceed. The first method is easier, but does not
work on all explorers (it does on moonscan), the second method is more involved,
but will always work. Try the first, and if it didn't work, then try the second.

## Constructor arguments, method #1

Leave the constructor arguments field empty, and just proceed to verification.
This step will fail, because the deployed bytecode will actually include the
constructor arguments, and the result of compiling the source file doesn't.
At this point, some explorers will just return a generic error message saying
the bytecodes didn't match, without any additional information. If this is the
case, go to method #2.

If the page shows what the *expected* bytecode was, and lists out the *actual*
bytecodes it found in the source file, then we may proceed here. The *expected*
bytecode will be the same as the `BridgeToken` bytecode with the constructor
arguments appended to the end. This means that the `BridgeToken` bytecode is a
proper prefix of the expected bytecode. Just copy the rest of the bytes of the
expected bytecode (for example in your favorite text editor), i.e. the suffix
that comes after the `BridgeToken` prefix. Go back to the previous page, and use
these bytes as the constructor arguments. This time, verification should succeed.

## Constructor arguments, method #2

If the explorer page does not show the expected and actual bytecodes, then we
need to compute the arguments ourselves. For this, we will need the `forge` tool
(install from https://getfoundry.sh/), and a local checkout of the wormhole repository

```sh
$ git clone https://github.com/wormhole-foundation/wormhole.git
```

First, install the dependencies by running the `make dependencies` in the
`ethereum` folder:

```sh
wormhole/ethereum $ make dependencies
```

Next, we will run a `forge` script to compute the constructor arguments. For
this, we will need two pieces of data. First, the address of the Portal token
bridge contract on the chain we're performing verification on. This can be found
in
https://github.com/wormhole-foundation/wormhole/blob/main/sdk/js/src/utils/consts.ts
under the MAINNET section (or TESTNET if you're verifying a testnet contract).
For our example, the mainnet moonbeam token bridge contract is
`0xb1731c586ca89a23809861c6103f0b96b3f57d92`.

Next, we'll need the VAA (the signed wormhole message) that was sent to the
token bridge contract to create the wrapped asset. This can be found by finding
the transaction that created this contract. On the contract's page
https://moonscan.io/address/0xab3f0245b83feb11d15aaffefd7ad465a59817ed, the
"Content creator" field has a link to the "at txn ...". In our case, this is
https://moonscan.io/tx/0x9f6db0a0749558e8ef5bf24c1be148498fb3c451d040198b7362c2ce32469e67

Expand the details, which shows a call to `createWrapped` in the "Input data"
section. Click "Decode input data" which will show the hex value of the
`encodedVM` argument. In our case it's

```
0x01000000020d00aa6b30da219a0573e1a2ff47b4e51b4768ff671ac99b4ca851b69a5c756c495f5e5416b1bacd1e65e01085184a0ea167bb8160db15621e39b079b7d8cae26e71000207a7e7d1053e143dffa3c3023ca7ca8816bcfa62d264112d3b51ca4eadde5cb8615673252063c9bc327ba5eaeeef37fa8e08064822b6de8746065606de2749b4010349718c3a2ed771b885b972d01ee36e9267b142945e8d4af06c7b6d43f12830f5745f17e70eb368f510520cd5386a44dd8d5c6f875f8e28c40f8ff8e559400358000418b99fc0b8090b4ec7c6cde6c43f82dc9f1c15a7610624833287c963bf776c7641b1a8410acb4f1b927c3784dd9290aaa50fca818c728d33e42de9b16b3d3dba0005fb5fce6005369257755550e4e6f7b1023fab3f83e4e70ec822ade40571fb712b26e537b7394278151055048d85553c73c1c7dbb65a956c05ded272b7c480d0c001066732b0cd475f74d2afacee7ce175a44648f2cc0ffc88f07b29edd7bd6497b9c728bd7e2df8f4bd926e0faa2820874d2a9006e7b8bb7fefd4876ba7acebfb4271000738db4897b6c4054c4c724254d02c6dfe2486379d4ee8ff3ccc4e180395d798784ca1d35a219e34c31ae3defadc0765aca0e81b047f5748663164cf2060e33465010c03b1fd13f924d56238344d6a9de1d650120b106e93f5638b2eb23fa7706b9ecb1e2fa10c805e6ec4f24742c7e43fb815084dcb18720fb6eae90d59c1824acf64010d13c022cfea6df258340a44b7b5667da9f6c739d92b2f77351eeb0217faf244a84d50be71579b65850f92397deaa133e267f224ea2537c223a7c9966417843917000fdc31f1540f66fa5d2991b52af617be24eaa66f41ad5542ac0b4f972b0afafcec3e57f52e4e71b933fad4ec9cf69a4c4bdbbb1a2a9b44f07a287be493dcb1fdb50010904167e5728f1405aceaacb67832e23ebe374f6bda35f75e9bb79c21acbda8e50ad864f24a9fd26f0f6c1a4eba02506db0669a8cd9829245b40c4510aa0bd74f001104877f08d3917e176ac487b25e03d123121171716387a391b56053eaed9364b42becc817ef9973fdeb117e75dd665a7b22a7641c60fdb8786a50082a463702760112739bc4983ac0408cbc13969b7ba6715dfef337324d39b36f8120e883584f51726ea0f07a4e08d66dd5f1144d7dbb49720cd0547834ee58d5b918d242e956eb720163396ad38785000000020000000000000000000000003ee18b2214aff97000d974cf647e7c347e8fa5850000000000014c530102000000000000000000000000c02aaa39b223fe8d0a0e5c4f27ead9083c756cc200021257455448000000000000000000000000000000000000000000000000000000005772617070656420457468657200000000000000000000000000000000000000
```

We copy this field and go back to our terminal. Run the following

```
wormhole/ethereum $ forge script scripts/TokenABI.s.sol -s "token_constructor_args(bytes, address)" <VAA-BYTES> <CONTRACT-ADDRESS>
```

where in place of `<VAA-BYTES>`, substitute the hex sequence we just copied from
the explorer, and in place of `<CONTRACT-ADDRESS>`, the token bridge's address
from before (`0xb1731c586ca89a23809861c6103f0b96b3f57d92` for moonbeam).

Running that command prints the following:

```
0x000000000000000000000000b1731c586ca89a23809861c6103f0b96b3f57d9200000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000164c71f461500000000000000000000000000000000000000000000000000000000000000e0000000000000000000000000000000000000000000000000000000000000012000000000000000000000000000000000000000000000000000000000000000120000000000000000000000000000000000000000000000000000000000014c53000000000000000000000000b1731c586ca89a23809861c6103f0b96b3f57d920000000000000000000000000000000000000000000000000000000000000002000000000000000000000000c02aaa39b223fe8d0a0e5c4f27ead9083c756cc2000000000000000000000000000000000000000000000000000000000000000d57726170706564204574686572000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000004574554480000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000
```

Copy the hex *excluding* the 0x at the front, and paste that into the
constructor arguments field, then hit verify. The contract should now be verified.
