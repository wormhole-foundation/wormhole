import "dotenv/config";
import * as os from "os"
import { SigningCosmWasmClient, InstantiateResult } from "@cosmjs/cosmwasm-stargate";
import { GasPrice } from "@cosmjs/stargate"
import { Secp256k1HdWallet } from "@cosmjs/amino";
import { MsgExecuteContract } from "cosmjs-types/cosmwasm/wasm/v1/tx";
import * as fs from "fs";
import { readdirSync, } from "fs";
import * as util from 'util'
import { Bech32, toHex, toUtf8, fromBase64 } from "@cosmjs/encoding";
import { zeroPad, keccak256 } from "ethers/lib/utils.js";
import * as elliptic from "elliptic"

import * as devnetConsts from "./devnet-consts.json"
import { concatArrays, encodeUint8 } from "./utils";

if (process.env.INIT_SIGNERS === "undefined") {
  let msg = `.env is missing. run "make contracts-tools-deps" to fetch.`
  console.error(msg)
  throw msg
}

const readFileAsync = util.promisify(fs.readFile);


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
const ARTIFACTS_PATH = "../artifacts/"
/* Check that the artifact folder contains all the wasm files we expect and nothing else */

try {
  const actual_artifacts = readdirSync(ARTIFACTS_PATH).filter((a) =>
    a.endsWith(".wasm")
  );

  const missing_artifacts = artifacts.filter(
    (a) => !actual_artifacts.includes(a)
  );
  if (missing_artifacts.length) {
    console.log(
      "Error during wormchain deployment. The following files are expected to be in the artifacts folder:"
    );
    missing_artifacts.forEach((file) => console.log(`  - ${file}`));
    console.log(
      "Hint: the deploy script needs to run after the contracts have been built."
    );
    console.log(
      "External binary blobs need to be manually added in tools/Dockerfile."
    );
    process.exit(1);
  }
} catch (err) {
  console.error(`${ARTIFACTS_PATH} cannot be read. Do you need to run "make contracts-deploy-setup"?`)
  process.exit(1)
}



async function main() {

  /* Set up cosmos client & wallet */

  let host = devnetConsts.chains[3104].tendermintUrlLocal
  if (os.hostname().includes("wormchain-deploy")) {
    // running in tilt devnet
    host = devnetConsts.chains[3104].tendermintUrlTilt
  }
  const denom = devnetConsts.chains[3104].addresses.native.denom
  const mnemonic = devnetConsts.chains[3104].accounts.wormchainNodeOfGuardian0.mnemonic
  const addressPrefix = "wormhole"
  const signerPk = String(process.env.TEST_SIGNER_PK)

  const w = await Secp256k1HdWallet.fromMnemonic(mnemonic, { prefix: addressPrefix })


  const gas = GasPrice.fromString(`0${denom}`)
  let cwc: SigningCosmWasmClient
  try {
    cwc = await SigningCosmWasmClient.connectWithSigner(host, w, { prefix: addressPrefix, gasPrice: gas })
  } catch (e) {
    let msg = `could not connect to wormchain host: ${host}`
    if (e?.message) {
      console.error(e.message)
    }
    throw msg
  }


  // there are several Cosmos chains in devnet, so check the config is as expected
  let id = await cwc.getChainId()
  if (id !== "wormchain") {
    throw new Error(`Wormchain CosmWasmClient connection produced an unexpected chainID: ${id}`)
  }

  const signers = await w.getAccounts()
  const signer = signers[0].address
  console.log("wormchain contract deployer is: ", signer)

  /* Deploy artifacts */

  const codeIds: { [name: ContractName]: number } = await artifacts.reduce(async (prev, file) => {
    // wait for the previous to finish, to avoid the race condition of wallet sequence mismatch.
    const accum = await prev

    const contract_bytes = await readFileAsync(`${ARTIFACTS_PATH}${file}`);

    const i = await cwc.upload(signer, contract_bytes, "auto", "")
    console.log(`uploaded ${file}, codeID: ${i.codeId}, tx: ${i.transactionHash}`, i.codeId, i.transactionHash)

    accum[file] = i.codeId
    return accum
  }, Object())

  /* Instantiate contracts.
   *
   * We instantiate the core contracts here (i.e. wormhole itself and the bridge contracts).
   * The wrapped asset contracts don't need to be instantiated here, because those
   * will be instantiated by the on-chain bridge contracts on demand.
   * */

  // Governance constants defined by the Wormhole spec.
  const govChain = 1;
  const govAddress =
    "0000000000000000000000000000000000000000000000000000000000000004";

  async function instantiate(code_id: number, inst_msg: any, label: string) {
    try {
      let inst = await cwc.instantiate(signer, code_id, inst_msg, label, "auto", {})
      let addr = inst.contractAddress
      let txHash = inst.transactionHash
      console.log(`deployed contract ${label}, codeID: ${code_id}, address: ${addr}, txHash: ${txHash}`)

      return addr
    }
    catch (e) {
      console.error(`failed instantiating ${label}. error: `, e)
      throw e
    }
  }

  // Instantiate contracts.
  // NOTE: Only append at the end, the ordering must be deterministic.

  const addresses: { [contractName: string]: InstantiateResult["transactionHash"] } = {};

  const init_guardians: string[] = JSON.parse(String(process.env.INIT_SIGNERS));
  if (!init_guardians || init_guardians.length === 0) {
    throw "failed to get initial guardians from .env file.";
  }

  console.log("going to instantiate wormhole core")
  addresses["wormhole.wasm"] = await instantiate(
    codeIds["wormhole.wasm"],
    {
      gov_chain: govChain,
      gov_address: Buffer.from(govAddress, "hex").toString("base64"),
      guardian_set_expirity: 864000000,
      initial_guardian_set: {
        addresses: init_guardians.map((hex: string) => {
          return {
            bytes: Buffer.from(hex, "hex").toString("base64"),
          };
        }),
        expiration_time: 0,
      },
      chain_id: WORMCHAIN_ID,
      fee_denom: "uworm",
    },
    "wormhole"
  );
  console.log("done instantiating wormhole core")

  console.log("going to instantiate token bridge")
  const tb_inst = {
    chain_id: WORMCHAIN_ID,
    gov_chain: govChain,
    gov_address: Buffer.from(govAddress, "hex").toString("base64"),
    wormhole_contract: "", // addresses["wormhole.wasm"],
    wrapped_asset_code_id: codeIds["cw20_wrapped_2.wasm"],
    native_denom: "uworm",
    native_symbol: "WORM",
    native_decimals: 6,
  }
  addresses["token_bridge_terra_2.wasm"] = await instantiate(
    codeIds["token_bridge_terra_2.wasm"],
    tb_inst,
    "tokenBridge"
  );
  console.log("Wormchain TokenBridge address in wormhole (hex) format",
    convert_address_to_hex(addresses["token_bridge_terra_2.wasm"]))
  console.log("done instantiating token bridge")


  console.log("going to instantiate mock cw20")
  addresses["mock.wasm"] = await instantiate(
    codeIds["cw20_base.wasm"],
    {
      name: "MOCK",
      symbol: "MCK",
      decimals: 6,
      initial_balances: [
        {
          address: signer,
          amount: "100000000",
        },
      ],
      mint: null,
    },
    "mock"
  );
  console.log("done instantiating mock cw20 ")


  /* Registrations: tell the bridge contracts to know about each other */


  const registrations: { [chainName: string]: string } = {
    // keys are only used for logging success/failure
    "solana": String(process.env.REGISTER_SOL_TOKEN_BRIDGE_VAA),
    "ethereum": String(process.env.REGISTER_ETH_TOKEN_BRIDGE_VAA),
    "bsc": String(process.env.REGISTER_BSC_TOKEN_BRIDGE_VAA),
    "algo": String(process.env.REGISTER_ALGO_TOKEN_BRIDGE_VAA),
    "terra": String(process.env.REGISTER_TERRA_TOKEN_BRIDGE_VAA),
    "near": String(process.env.REGISTER_NEAR_TOKEN_BRIDGE_VAA),
    "terra2": String(process.env.REGISTER_TERRA2_TOKEN_BRIDGE_VAA),
    "aptos": String(process.env.REGISTER_APTOS_TOKEN_BRIDGE_VAA),
  }


  console.log("going to send token bridge registrations")
  for (let chain in registrations) {
    const body = { data: Buffer.from(registrations[chain], "hex").toString("base64") };
    const msg = {
      typeUrl: "/cosmwasm.wasm.v1.MsgExecuteContract",
      value: MsgExecuteContract.fromPartial({
        sender: signer,
        contract: addresses["token_bridge_terra_2.wasm"],
        msg: toUtf8(JSON.stringify({
          submit_vaa: body
        }))
      })
    }
    const res = await cwc.signAndBroadcast(signer, [msg], "auto")
    console.log(`sent token bridge registration for ${chain}, tx: `, res.transactionHash);
  }
  console.log("done sending token bridge registrations")


  console.log("going to instantiate accounting")
  const instantiateMsg = {}
  addresses["wormchain_accounting.wasm"] = await instantiate(
    codeIds["wormchain_accounting.wasm"],
    instantiateMsg,
    "wormchainAccounting"
  )
  console.log("done instantiating accounting")


  console.log("going to send accounting registrations")
  const accountingRegistrations = Object.values(registrations)
    .map(r => Buffer.from(r, "hex").toString("base64"))

  const msg = {
    typeUrl: "/cosmwasm.wasm.v1.MsgExecuteContract",
    value: MsgExecuteContract.fromPartial({
      sender: signer,
      contract: addresses["wormchain_accounting.wasm"],
      msg: toUtf8(JSON.stringify({
        submit_v_a_as: {
          vaas: accountingRegistrations,
        }
      }))
    })
  }
  const res = await cwc.signAndBroadcast(signer, [msg], "auto");
  console.log(`sent accounting chain registrations, tx: `, res.transactionHash);
  console.log("done sending accounting registrations")


  // Terra addresses are "human-readable", but for cross-chain registrations, we
  // want the "canonical" version
  function convert_address_to_hex(human_addr: string) {
    return "0x" + toHex(zeroPad(Bech32.decode(human_addr).data, 32));
  }

}

try {
  main()
} catch (e: any) {
  if (e?.message) {
    console.error(e.message)
  }
  throw e
}
