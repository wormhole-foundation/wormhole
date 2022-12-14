import { Secp256k1HdWallet } from "@cosmjs/amino";
import { SigningCosmWasmClient, toBinary } from "@cosmjs/cosmwasm-stargate";
import { GasPrice } from "@cosmjs/stargate";
import "dotenv/config";

// generated types
import {
  Account,
  ExecuteMsg,
  InstantiateMsg,
  Modification,
  Signature,
  Transfer,
  Observation,
} from "./client/WormchainAccounting.types";

import { keccak256 } from "@cosmjs/crypto";
import { Bech32, fromBase64 } from "@cosmjs/encoding";
import * as elliptic from "elliptic";
import { zeroPad } from "ethers/lib/utils.js";

async function main() {
  /* Set up cosmos client & wallet */

  // const host = "http://0.0.0.0:26657"   // TODO - make this 26659 for tilt
  const host = "http://localhost:26659";
  const addressPrefix = "wormhole";
  const denom = "uworm";
  const mnemonic =
    "notice oak worry limit wrap speak medal online prefer cluster roof addict wrist behave treat actual wasp year salad speed social layer crew genius";
  const TEST_SIGNER_PK =
    "cfb12303a19cde580bb4dd771639b0d26bc68353645571a8cff516ab2ee113a0";

  const w = await Secp256k1HdWallet.fromMnemonic(mnemonic, {
    prefix: addressPrefix,
  });

  const gas = GasPrice.fromString(`0${denom}`);
  let cwc = await SigningCosmWasmClient.connectWithSigner(host, w, {
    prefix: addressPrefix,
    gasPrice: gas,
  });

  // there is no danger here, just several Cosmos chains in devnet, so check for config issues
  let id = await cwc.getChainId();
  if (id !== "wormchain") {
    throw new Error(
      `Wormchain CosmWasmClient connection produced an unexpected chainID: ${id}`
    );
  }

  const signers = await w.getAccounts();
  const signer = signers[0].address;
  console.log("wormchain contract deployer is: ", signer);

  // From the logs of deploy_accounting.ts
  const accountingAddress =
    "wormhole1466nf3zuxpya8q9emxukd7vftaf6h4psr0a07srl5zw74zh84yjq4lyjmh";

  const observations: Observation[] = [];

  // object to json string, then to base64 (serde binary)
  const observationsBinaryString = toBinary(observations);

  // base64 string to Uint8Array,
  // so we have bytes to work with for signing, though not sure 100% that's correct.
  const observationsBytes = fromBase64(observationsBinaryString);

  // create the "digest" for signing.
  // The contract will calculate the digest of the "data",
  // then use that with the signature to ec recover the publickey that signed.
  const digest = keccak256(keccak256(observationsBytes));

  const ec = new elliptic.ec("secp256k1");

  // create key from the devnet guardian0's private key
  const key = ec.keyFromPrivate(Buffer.from(TEST_SIGNER_PK, "hex"));

  // check the key
  const { result, reason } = key.validate();
  console.log("key validate result, reason, ", result, reason);

  // sign the digest
  const signature = key.sign(digest, { canonical: true });

  // create 65 byte signature (64 + 1)
  const signedParts = [
    zeroPad(signature.r.toBuffer(), 32),
    zeroPad(signature.s.toBuffer(), 32),
    encodeUint8(signature.recoveryParam || 0),
  ];

  // combine parts to be Uint8Array with length 65
  const signed = concatArrays(signedParts);
  console.log("signed.len ", signed.length);
  console.log("signed");
  console.log(signed);

  // try sending the instantiate message in a few different formats:

  // the message type is accepted, but the signature verificaton fails. error:
  // failed to execute message; message index: 0: failed to verify quorum:
  // Generic error: Querier contract error: codespace: wormhole, code: 1102: instantiate wasm contract failed

  // send the instantiate object as described by the generated TS client types
  const executeMsg: ExecuteMsg = {
    submit_observations: {
      observations: observationsBinaryString,
      guardian_set_index: 0,
      signature: {
        index: 0,
        signature: Array.from(signed) as Signature["signature"],
      },
    },
  };
  try {
    let inst = await cwc.execute(signer, accountingAddress, executeMsg, "auto");
    let txHash = inst.transactionHash;
    console.log(`executed contract txHash: ${txHash}`);
  } catch (err: any) {
    if (err?.message) {
      console.error(err.message);
    }
    // throw err
  }

  function zeroPadBytes(value: string, length: number) {
    while (value.length < 2 * length) {
      value = "0" + value;
    }
    return value;
  }

  function concatArrays(arrays: Uint8Array[]): Uint8Array {
    const totalLength = arrays.reduce((accum, x) => accum + x.length, 0);
    const result = new Uint8Array(totalLength);

    for (let i = 0, offset = 0; i < arrays.length; i++) {
      result.set(arrays[i], offset);
      offset += arrays[i].length;
    }

    return result;
  }
  function encodeUint8(value: number): Uint8Array {
    if (value >= 2 ** 8 || value < 0) {
      throw new Error(`Out of bound value in Uint8: ${value}`);
    }

    return new Uint8Array([value]);
  }
}

try {
  main();
} catch (e) {
  console.error(e);
  throw e;
}
