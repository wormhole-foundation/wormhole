01000000000100bc708675768ba853af2e01c6217317fbb68500a508a39fece15dac27d36c7c804cf94ad9e4b2134209bbdad7c9479d9929c60bda4612a74c61c06efebacc4da401000003cb5d690100000200000000000000000000000026b4afb60d6c903165150c6f0aa14f8016be4aec00000000000000010f01000000000000000000000000d1a269d9b0dfb66cfdaf89cf0c6e6f8df0615ad00002415045f09f9092000000000000000000000000000000000000000000000000004e6f7420616e2041504520f09f90920000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000a5268747470733a2f2f636c6f7564666c6172652d697066732e636f6d2f697066732f516d65536a53696e4870506e6d586d73704d6a776958794e367a533445397a63636172694752336a7863615774712f31301212121212121212121212121212121212121212121212121212121212121212000f

--

1) vaa is received for a user for a token where the user has not paid the storage deposit in that token
2) vaa gets submitted without using the SDK which detects this and correctly registers/pays-storage-deposit

choice 1:

   VAA gets consumed but the tokens don't get transfered

choice 2:

   The token bridge proactively throws money at the token, which it hopes will get refunded, 
   to pay for storage if it is not already there

   VAA gets consumed but the money might not get refunded... due to token implementor being an asshat




// demonstrates how to query the state without setting
// up an account. (View methods only)
const { providers } = require("near-api-js");
//network config (replace testnet with mainnet or betanet)
const provider = new providers.JsonRpcProvider("https://rpc.testnet.near.org");

getState();

async function getState() {
  const rawResult = await provider.query({
    request_type: "call_function",
    account_id: "guest-book.testnet",
    method_name: "getMessages",
    args_base64: "e30=",
    finality: "optimistic",
  });

  // format result
  const res = JSON.parse(Buffer.from(rawResult.result).toString());
  console.log(res);
}


import { Account as nearAccount } from "near-api-js";

My impression is:

   https://docs.near.org/docs/tutorials/near-indexer

   https://thewiki.near.page/events-api   

==
kubectl exec -it near-0 -c near-node -- /bin/bash

My NEAR notes so far...

If needed, install `Rust`:

  curl https://sh.rustup.rs -sSf | sh

You need at least version 1.56 or later

  rustup default 1.56
  rustup update
  rustup target add wasm32-unknown-unknown

If needed, install `near-cli`:

   npm install near-cli -g

To install the npm dependencies of this test program

   npm install

for the near sdk, we are dependent on 4.0.0 or later  (where the ecrecover API is)

  https://docs.rs/near-sdk/4.0.0/near_sdk/index.html
  near-sdk = { version = "4.0.0", features = ["unstable"] }

  This has been stuck into Cargo.toml

to bring up the sandbox, start a tmux window and run

  rm -rf _sandbox
  mkdir -p _sandbox
  near-sandbox --home _sandbox init
  near-sandbox --home _sandbox run

https://docs.near.org/docs/develop/contracts/sandbox

First thing, lets put this in a docker in Tilt..

vaa_verify?

near-sdk-rs/near-sdk/src/environment/env.rs: (still unstable)

    /// Recovers an ECDSA signer address from a 32-byte message `hash` and a corresponding `signature`
    /// along with `v` recovery byte.
    ///
    /// Takes in an additional flag to check for malleability of the signature
    /// which is generally only ideal for transactions.
    ///
    /// Returns 64 bytes representing the public key if the recovery was successful.
    #[cfg(feature = "unstable")]
    pub fn ecrecover(
        hash: &[u8],
        signature: &[u8],
        v: u8,
        malleability_flag: bool,
    ) -> Option<[u8; 64]> {
        unsafe {
            let return_code = sys::ecrecover(
                hash.len() as _,
                hash.as_ptr() as _,
                signature.len() as _,
                signature.as_ptr() as _,
                v as u64,
                malleability_flag as u64,
                ATOMIC_OP_REGISTER,
            );
            if return_code == 0 {
                None
            } else {
                Some(read_register_fixed_64(ATOMIC_OP_REGISTER))
            }
        }
    }

you can look for test_ecrecover()    in the same file...

When building the sandbox, it is on port 3030 and we will need access to the validator_key.json...

curl http://localhost:3031/validator_key.json

function getConfig(env) {
  switch (env) {
    case "sandbox":
    case "local":
      return {
        networkId: "sandbox",
        nodeUrl: "http://localhost:3030",
        masterAccount: "test.near",
        contractAccount: "wormhole.test.near",
        keyPath: "./_sandbox/validator_key.json",
      };
  }
}

   .function_call(
                b"new".to_vec(),
                ft,
                data.to_vec(),
                vaa.sequence,
            )
