import "dotenv/config";
import { SigningCosmWasmClient, InstantiateResult } from "@cosmjs/cosmwasm-stargate";
import { GasPrice } from "@cosmjs/stargate"
import { Secp256k1HdWallet } from "@cosmjs/amino";
import { InstantiateMsg as TokenBridgeInstantiateMsg } from "./client/TokenBridge.types"
import { TokenBridgeMessageComposer } from "./client/TokenBridge.message-composer";

import fs, { readdirSync, } from "fs";
import util from 'util'
import { toHex } from "@cosmjs/encoding";
import { zeroPad, } from "ethers/lib/utils.js";

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



async function main() {

  /* Set up cosmos client & wallet */

  const host = "http://0.0.0.0:26657"   // TODO - make this 26659 for tilt
  const addressPrefix = "wormhole"
  const denom = "uworm"
  const mnemonic = "notice oak worry limit wrap speak medal online prefer cluster roof addict wrist behave treat actual wasp year salad speed social layer crew genius"

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
    let inst = await cwc.instantiate(signer, code_id, inst_msg, label, "auto", {})
    let addr = inst.contractAddress
    let txHash = inst.transactionHash
    console.log(`deployed contract ${label}, codeID: ${code_id}, address: ${addr}, txHash: ${txHash}`)

    return addr
  }

  // Instantiate contracts.
  // NOTE: Only append at the end, the ordering must be deterministic.

  const addresses: { [contractName: string]: InstantiateResult["transactionHash"] } = {};

  const init_guardians: string[] = JSON.parse(String(process.env.INIT_SIGNERS));
  if (!init_guardians || init_guardians.length === 0) {
    throw "failed to get initial guardians from .env file.";
  }

  addresses["wormhole.wasm"] = await instantiate(
    codeIds["wormhole.wasm"],
    {
      gov_chain: govChain,
      gov_address: Buffer.from(govAddress, "hex").toString("base64"),
      guardian_set_expirity: 864000000,
      initial_guardian_set: {
        // pub bytes: Binary, // 20-byte addresses
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

  const tb_inst: TokenBridgeInstantiateMsg = {
    chain_id: WORMCHAIN_ID,
    gov_chain: govChain,
    gov_address: Buffer.from(govAddress, "hex").toString("base64"),
    wormhole_contract: addresses["wormhole.wasm"],
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


  const tbMsgComposer = new TokenBridgeMessageComposer(signer, addresses["token_bridge_terra_2.wasm"])
  for (let chain in registrations) {
    const body = { data: Buffer.from(registrations[chain], "hex").toString("base64") };
    const msg = tbMsgComposer.submitVaa(body)
    const res = await cwc.signAndBroadcast(signer, [msg], "auto")
    console.log(`sent token bridge registration for ${chain}, tx: `, res.transactionHash);
  }

  console.log('done!')

}

try {
  main()
} catch (e) {
  console.error(e)
  throw e
}
