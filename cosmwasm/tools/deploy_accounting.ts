import "dotenv/config";
import { SigningCosmWasmClient, toBinary, setupWasmExtension, fromBinary } from "@cosmjs/cosmwasm-stargate";
import { GasPrice } from "@cosmjs/stargate"
import { Secp256k1HdWallet } from "@cosmjs/amino";
import { toUtf8 } from "@cosmjs/encoding";

// generated types
import {
  Binary, InstantiateMsg, Signature, ExecuteMsg, QueryMsg, TokenAddress, Uint256, Key, Transfer, Data, Balance,
  AllAccountsResponse, Account, Kind, AllModificationsResponse, Modification, AllPendingTransfersResponse, PendingTransfer,
  Observation, AllTransfersResponse, Empty
} from "./client/WormchainAccounting.types";

import { InstantiateMsg as TokenBridgeInstantiateMsg } from "./client/TokenBridge.types"
import { TokenBridgeMessageComposer } from "./client/TokenBridge.message-composer";
import { keccak256 } from "@cosmjs/crypto"
import { Bech32, toBase64, fromHex, toHex, fromBase64 } from "@cosmjs/encoding";
import { zeroPad, keccak256 as ethersKeccak256 } from "ethers/lib/utils.js";
import * as elliptic from "elliptic"
import { WormchainAccountingInterface } from "./client/WormchainAccounting.client";

/*
  NOTE: Only append to this array: keeping the ordering is crucial, as the
  contracts must be imported in a deterministic order so their addresses remain
  deterministic.
*/
type ContractName = string
const artifacts: ContractName[] = [
  "wormhole.wasm",
  "token_bridge_terra_2.wasm",
  "cw20_wrapped_2.wasm",
  "cw20_base.wasm",
  "mock_bridge_integration_2.wasm",
  // "shutdown_core_bridge_cosmwasm.wasm",
  // "shutdown_token_bridge_cosmwasm.wasm",
  "wormchain_accounting.wasm",
];

const WORMCHAIN_ID = 3104

/* Check that the artifact folder contains all the wasm files we expect and nothing else */

async function main() {

  /* Set up cosmos client & wallet */

  const host = "http://0.0.0.0:26657"   // TODO - make this 26659 for tilt
  const addressPrefix = "wormhole"
  const denom = "uworm"
  const mnemonic = "notice oak worry limit wrap speak medal online prefer cluster roof addict wrist behave treat actual wasp year salad speed social layer crew genius"
  const TEST_SIGNER_PK = "cfb12303a19cde580bb4dd771639b0d26bc68353645571a8cff516ab2ee113a0"

  const w = await Secp256k1HdWallet.fromMnemonic(mnemonic, { prefix: addressPrefix })

  const gas = GasPrice.fromString(`0${denom}`)
  let cwc = await SigningCosmWasmClient.connectWithSigner(host, w, { prefix: addressPrefix, gasPrice: gas })

  // there is no danger here, just several Cosmos chains in devnet, so check for config issues
  let id = await cwc.getChainId()
  if (id !== "wormchain") {
    throw new Error(`Wormchain CosmWasmClient connection produced an unexpected chainID: ${id}`)
  }

  const signers = await w.getAccounts()
  const signer = signers[0].address
  console.log("wormchain contract deployer is: ", signer)

  // Instantiate accounting

  // From the logs of deploy_wormchain.ts
  const tokenBridgeAddress = "wormhole1nc5tatafv6eyq7llkr2gv50ff9e22mnf70qgjlv737ktmt4eswrq0kdhcj"
  const accountingCodeId = 6


  async function instantiate(code_id: number, inst_msg: any, label: string) {
    // maybe try manually making a message and sending it? :
    // typeUrl: "/cosmwasm.wasm.v1.MsgInstantiateContract";
    let inst = await cwc.instantiate(signer, code_id, inst_msg, label, "auto")
    let addr = inst.contractAddress
    let txHash = inst.transactionHash
    console.log(`deployed contract ${label}, codeID: ${code_id}, address: ${addr}, txHash: ${txHash}`)

    return addr
  }



  // not sure what format the token bridge address should be in before it the entire message
  // is serialized to json + base64, by the @cosmjs/cosmwasm-stargate lib.
  // const formattedTB = Buffer.from(tokenBridgeAddress, "utf-8").toString("hex")  // 134 chars, 66 bytes
  // const formattedTB = convert_terra_address_to_hex(tokenBridgeAddress)
  const formattedTBAddr = zeroPad(Bech32.decode(tokenBridgeAddress).data, 32)
  // console.log("formattedTBAddr", formattedTBAddr)

  const accounts: Account[] = []
  const transfers: Transfer[] = []
  const modifications: Modification[] = []

  // create the object that will be the "data" that gets signed
  // the generated type "InstantiateMsg" says there is no
  // tokenbridge_addr prop on the object, but there is, so dont use the type I guess.
  const instantiateBody = {
    // not sure what is the correct format of token bridge address
    // tokenbridge_addr: formattedTBAddr,  // address as Uint8Array
    tokenbridge_addr: tokenBridgeAddress,  // address as native string (Bech32 with prefix)
    accounts,
    transfers,
    modifications,
  }

  // object to json string, then to base64 (serde binary)
  const instantiateBodyBinaryString = toBinary(instantiateBody)

  // base64 string to Uint8Array,
  // so we have bytes to work with for signing, though not sure 100% that's correct.
  const instantiateBodyBytes = fromBase64(instantiateBodyBinaryString)

  // create the "digest" for signing.
  // The contract will calculate the digest of the "data",
  // then use that with the signature to ec recover the publickey that signed.
  const digest = keccak256(keccak256(instantiateBodyBytes))



  const ec = new elliptic.ec("secp256k1");

  // create key from the devnet guardian0's private key
  const key = ec.keyFromPrivate(Buffer.from(TEST_SIGNER_PK, "hex"));

  // check the key
  const { result, reason } = key.validate()
  console.log("key validate result, reason, ", result, reason)

  // sign the digest
  const signature = key.sign(digest, { canonical: true });

  // create 65 byte signature (64 + 1)
  const signedParts = [
    zeroPad(signature.r.toBuffer(), 32),
    zeroPad(signature.s.toBuffer(), 32),
    encodeUint8(signature.recoveryParam || 0)
  ]

  // combine parts to be Uint8Array with length 65
  const signed = concatArrays(signedParts)
  console.log("signed.len ", signed.length)
  console.log("signed")
  console.log(signed)


  // try sending the instantiate message in a few different formats:


  // the message type is accepted, but the signature verificaton fails. error:
  // failed to execute message; message index: 0: failed to verify quorum:
  // Generic error: Querier contract error: codespace: wormhole, code: 1102: instantiate wasm contract failed

  // send the instantiate object as described by the generated TS client types
  const instantiateMsg: InstantiateMsg = {
    guardian_set_index: 0,
    instantiate: instantiateBodyBinaryString,
    signatures: [{
      index: 0,
      signature: Array.from(signed) as Signature["signature"]
    }]
  }
  try {
    const accountingAddress = await instantiate(
      accountingCodeId,
      instantiateMsg,
      "wormchainAccountingTyped"
    );
    console.log("done deployting Accounting! ", accountingAddress)
  } catch (err: any) {
    if (err?.message) {
      console.error(err.message)
    }
    // throw err
  }

  // this fails with:
  // Error parsing into type wormchain_accounting::msg::InstantiateMsg: Invalid type

  // try leaving it as an object, rather than serializing parts.
  const acctInstRaw = {
    guardian_set_index: 0,
    instantiate: instantiateBody,
    signatures: [{
      index: 0,
      signature: Array.from(signed)
    }]
  }
  try {
    const accountingAddress = await instantiate(
      accountingCodeId,
      acctInstRaw,
      "wormchainAccountingRaw"
    );
    console.log("done deployting Accounting! ", accountingAddress)
  } catch (err: any) {
    if (err?.message) {
      console.error(err.message)
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
    const totalLength = arrays.reduce((accum, x) => accum + x.length, 0)
    const result = new Uint8Array(totalLength)

    for (let i = 0, offset = 0; i < arrays.length; i++) {
      result.set(arrays[i], offset)
      offset += arrays[i].length
    }

    return result
  }
  function encodeUint8(value: number): Uint8Array {
    if (value >= 2 ** 8 || value < 0) {
      throw new Error(`Out of bound value in Uint8: ${value}`)
    }

    return new Uint8Array([value])
  }

}

try {
  main()
} catch (e) {
  console.error(e)
  throw e
}
