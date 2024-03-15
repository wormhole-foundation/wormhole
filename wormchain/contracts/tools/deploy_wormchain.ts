import {
  CHAIN_ID_WORMCHAIN,
  hexToUint8Array,
  Other,
  Payload,
  serialiseVAA,
  sign,
  VAA,
} from "@certusone/wormhole-sdk";
import { toBinary } from "@cosmjs/cosmwasm-stargate";
import { fromBase64, toUtf8, fromBech32 } from "@cosmjs/encoding";
import {
  getWallet,
  getWormchainSigningClient,
} from "@wormhole-foundation/wormchain-sdk";
import { ZERO_FEE } from "@wormhole-foundation/wormchain-sdk/lib/core/consts";
import "dotenv/config";
import * as fs from "fs";
import { readdirSync } from "fs";
import { keccak256 } from "js-sha3";
import * as os from "os";
import * as util from "util";
import * as devnetConsts from "./devnet-consts.json";

if (process.env.INIT_SIGNERS_KEYS_CSV === "undefined") {
  let msg = `.env is missing. run "make contracts-tools-deps" to fetch.`;
  console.error(msg);
  throw msg;
}

const init_guardians = JSON.parse(process.env.INIT_SIGNERS);
if (!init_guardians || init_guardians.length === 0) {
  throw "failed to get initial guardians from .env file.";
}

const VAA_SIGNERS = process.env.INIT_SIGNERS_KEYS_CSV.split(",");
const GOVERNANCE_CHAIN = Number(devnetConsts.global.governanceChainId);
const GOVERNANCE_EMITTER = devnetConsts.global.governanceEmitterAddress;

const readFileAsync = util.promisify(fs.readFile);

/*
  NOTE: Only append to this array: keeping the ordering is crucial, as the
  contracts must be imported in a deterministic order so their addresses remain
  deterministic.
*/
type ContractName = string;
const artifacts: ContractName[] = [
  "global_accountant.wasm",
  "wormchain_ibc_receiver.wasm",
  "ntt_global_accountant.wasm",
  "cw_wormhole.wasm",
  "cw_token_bridge.wasm",
  "cw20_wrapped_2.wasm",
  "ibc_translator.wasm",
];

// Governance constants defined by the Wormhole spec.
const govChain = 1;
const govAddress =
  "0000000000000000000000000000000000000000000000000000000000000004";

const ARTIFACTS_PATH = "../artifacts/";
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
  console.error(
    `${ARTIFACTS_PATH} cannot be read. Do you need to run "make contracts-deploy-setup"?`
  );
  process.exit(1);
}

async function main() {
  /* Set up cosmos client & wallet */

  let host = devnetConsts.chains[3104].tendermintUrlLocal;
  if (os.hostname().includes("wormchain-deploy")) {
    // running in tilt devnet
    host = devnetConsts.chains[3104].tendermintUrlTilt;
  }

  const mnemonic =
    devnetConsts.chains[3104].accounts.wormchainNodeOfGuardian0.mnemonic;

  const wallet = await getWallet(mnemonic);
  const client = await getWormchainSigningClient(host, wallet);

  // there are several Cosmos chains in devnet, so check the config is as expected
  let id = await client.getChainId();
  if (id !== "wormchain") {
    throw new Error(
      `Wormchain CosmWasmClient connection produced an unexpected chainID: ${id}`
    );
  }

  const signers = await wallet.getAccounts();
  const signer = signers[0].address;
  console.log("wormchain contract deployer is: ", signer);

  /* Deploy artifacts */

  const codeIds: { [name: ContractName]: number } = await artifacts.reduce(
    async (prev, file) => {
      // wait for the previous to finish, to avoid the race condition of wallet sequence mismatch.
      const accum = await prev;

      const contract_bytes = await readFileAsync(`${ARTIFACTS_PATH}${file}`);

      const payload = keccak256(contract_bytes);
      let vaa: VAA<Other> = {
        version: 1,
        guardianSetIndex: 0,
        signatures: [],
        timestamp: 0,
        nonce: 0,
        emitterChain: GOVERNANCE_CHAIN,
        emitterAddress: GOVERNANCE_EMITTER,
        sequence: BigInt(Math.floor(Math.random() * 100000000)),
        consistencyLevel: 0,
        payload: {
          type: "Other",
          hex: `0000000000000000000000000000000000000000005761736D644D6F64756C65010${CHAIN_ID_WORMCHAIN.toString(
            16
          )}${payload}`,
        },
      };
      vaa.signatures = sign(VAA_SIGNERS, vaa as unknown as VAA<Payload>);
      console.log("uploading", file);
      const msg = client.core.msgStoreCode({
        signer,
        wasm_byte_code: new Uint8Array(contract_bytes),
        vaa: hexToUint8Array(serialiseVAA(vaa as unknown as VAA<Payload>)),
      });
      const result = await client.signAndBroadcast(signer, [msg], {
        ...ZERO_FEE,
        gas: "10000000",
      });
      const codeId = Number(
        JSON.parse(result.rawLog)[0]
          .events.find(({ type }) => type === "store_code")
          .attributes.find(({ key }) => key === "code_id").value
      );
      console.log(
        `uploaded ${file}, codeID: ${codeId}, tx: ${result.transactionHash}`
      );

      accum[file] = codeId;
      return accum;
    },
    Object()
  );

  // Instantiate contracts.

  async function instantiate(code_id: number, inst_msg: any, label: string) {
    const instMsgBinary = toBinary(inst_msg);
    const instMsgBytes = fromBase64(instMsgBinary);

    // see /sdk/vaa/governance.go
    const codeIdBuf = Buffer.alloc(8);
    codeIdBuf.writeBigInt64BE(BigInt(code_id));
    const codeIdHash = keccak256(codeIdBuf);
    const codeIdLabelHash = keccak256(
      Buffer.concat([
        Buffer.from(codeIdHash, "hex"),
        Buffer.from(label, "utf8"),
      ])
    );
    const fullHash = keccak256(
      Buffer.concat([Buffer.from(codeIdLabelHash, "hex"), instMsgBytes])
    );

    console.log(fullHash);

    let vaa: VAA<Other> = {
      version: 1,
      guardianSetIndex: 0,
      signatures: [],
      timestamp: 0,
      nonce: 0,
      emitterChain: GOVERNANCE_CHAIN,
      emitterAddress: GOVERNANCE_EMITTER,
      sequence: BigInt(Math.floor(Math.random() * 100000000)),
      consistencyLevel: 0,
      payload: {
        type: "Other",
        hex: `0000000000000000000000000000000000000000005761736D644D6F64756C65020${CHAIN_ID_WORMCHAIN.toString(
          16
        )}${fullHash}`,
      },
    };
    // TODO: check for number of guardians in set and use the corresponding keys
    vaa.signatures = sign(VAA_SIGNERS, vaa as unknown as VAA<Payload>);
    const msg = client.core.msgInstantiateContract({
      signer,
      code_id,
      label,
      msg: instMsgBytes,
      vaa: hexToUint8Array(serialiseVAA(vaa as unknown as VAA<Payload>)),
    });
    const result = await client.signAndBroadcast(signer, [msg], {
      ...ZERO_FEE,
      gas: "10000000",
    });
    console.log("contract instantiation msg: ", msg);
    console.log("contract instantiation result: ", result);
    const addr = JSON.parse(result.rawLog)[0]
      .events.find(({ type }) => type === "instantiate")
      .attributes.find(({ key }) => key === "_contract_address").value;
    console.log(
      `deployed contract ${label}, codeID: ${code_id}, address: ${addr}, txHash: ${result.transactionHash}`
    );

    return addr;
  }

  // Instantiate contracts.
  // NOTE: Only append at the end, the ordering must be deterministic.

  const addresses: {
    [contractName: string]: string;
  } = {};

  const registrations: { [chainName: string]: string } = {
    // keys are only used for logging success/failure
    solana: String(process.env.REGISTER_SOL_TOKEN_BRIDGE_VAA),
    ethereum: String(process.env.REGISTER_ETH_TOKEN_BRIDGE_VAA),
    bsc: String(process.env.REGISTER_BSC_TOKEN_BRIDGE_VAA),
    algo: String(process.env.REGISTER_ALGO_TOKEN_BRIDGE_VAA),
    terra: String(process.env.REGISTER_TERRA_TOKEN_BRIDGE_VAA),
    near: String(process.env.REGISTER_NEAR_TOKEN_BRIDGE_VAA),
    terra2: String(process.env.REGISTER_TERRA2_TOKEN_BRIDGE_VAA),
    aptos: String(process.env.REGISTER_APTOS_TOKEN_BRIDGE_VAA),
    sui: String(process.env.REGISTER_SUI_TOKEN_BRIDGE_VAA),
  };

  const instantiateMsg = {};
  addresses["global_accountant.wasm"] = await instantiate(
    codeIds["global_accountant.wasm"],
    instantiateMsg,
    "wormchainAccounting"
  );
  console.log("instantiated accounting: ", addresses["global_accountant.wasm"]);

  const accountingRegistrations = Object.values(registrations).map((r) =>
    Buffer.from(r, "hex").toString("base64")
  );
  const msg = client.wasm.msgExecuteContract({
    sender: signer,
    contract: addresses["global_accountant.wasm"],
    msg: toUtf8(
      JSON.stringify({
        submit_vaas: {
          vaas: accountingRegistrations,
        },
      })
    ),
    funds: [],
  });
  const res = await client.signAndBroadcast(signer, [msg], {
    ...ZERO_FEE,
    gas: "10000000",
  });
  console.log(`sent accounting chain registrations, tx: `, res.transactionHash);

  const wormchainIbcReceiverInstantiateMsg = {};
  addresses["wormchain_ibc_receiver.wasm"] = await instantiate(
    codeIds["wormchain_ibc_receiver.wasm"],
    wormchainIbcReceiverInstantiateMsg,
    "wormchainIbcReceiver"
  );
  console.log(
    "instantiated wormchain ibc receiver contract: ",
    addresses["wormchain_ibc_receiver.wasm"]
  );

  // Generated VAA using
  // `guardiand template ibc-receiver-update-channel-chain --channel-id channel-0 --chain-id 32 --target-chain-id 3104 > wormchain.prototxt`
  // `guardiand admin governance-vaa-verify wormchain.prototxt`
  let wormchainIbcReceiverWhitelistVaa: VAA<Other> = {
    version: 1,
    guardianSetIndex: 0,
    signatures: [],
    timestamp: 0,
    nonce: 0,
    emitterChain: GOVERNANCE_CHAIN,
    emitterAddress: GOVERNANCE_EMITTER,
    sequence: BigInt(Math.floor(Math.random() * 100000000)),
    consistencyLevel: 0,
    payload: {
      type: "Other",
      hex: `0000000000000000000000000000000000000000004962635265636569766572010c20000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000006368616e6e656c2d300020`,
    },
  };
  wormchainIbcReceiverWhitelistVaa.signatures = sign(
    VAA_SIGNERS,
    wormchainIbcReceiverWhitelistVaa as unknown as VAA<Payload>
  );
  const wormchainIbcReceiverUpdateWhitelistMsg = {
    submit_update_channel_chain: {
      vaas: [
        Buffer.from(
          serialiseVAA(
            wormchainIbcReceiverWhitelistVaa as unknown as VAA<Payload>
          ),
          "hex"
        ).toString("base64"),
      ],
    },
  };
  const executeMsg = client.wasm.msgExecuteContract({
    sender: signer,
    contract: addresses["wormchain_ibc_receiver.wasm"],
    msg: toUtf8(JSON.stringify(wormchainIbcReceiverUpdateWhitelistMsg)),
    funds: [],
  });
  const updateIbcWhitelistRes = await client.signAndBroadcast(
    signer,
    [executeMsg],
    {
      ...ZERO_FEE,
      gas: "10000000",
    }
  );
  console.log(
    "updated wormchain_ibc_receiver whitelist: ",
    updateIbcWhitelistRes.transactionHash,
    updateIbcWhitelistRes.code
  );

  const nttGlobalAccountantInstantiateMsg = {};
  addresses["ntt_global_accountant.wasm"] = await instantiate(
    codeIds["ntt_global_accountant.wasm"],
    nttGlobalAccountantInstantiateMsg,
    "wormchainNTTAccounting"
  );
  console.log(
    "instantiated NTT accounting: ",
    addresses["ntt_global_accountant.wasm"]
  );

  const allowListResponse = await client.signAndBroadcast(
    signer,
    [
      client.core.msgCreateAllowlistEntryRequest({
        signer: signer,
        address: "wormhole14vtqhv6550uh6gycxxum8qmx3kmy7ak2qwzecx",
        name: "ibcRelayer",
      }),
      client.core.msgCreateAllowlistEntryRequest({
        signer: signer,
        address: "wormhole1s5a6dg9p902z5rhjgkk0ts8lulvtmhmpftasxe",
        name: "guardianGatewayRelayer0",
      }),
      client.core.msgCreateAllowlistEntryRequest({
        signer: signer,
        address: "wormhole1dtwappgz4zfmlhay44x5r787u6ap0zhrk2m09m",
        name: "guardianGatewayRelayer1",
      }),
      client.core.msgCreateAllowlistEntryRequest({
        signer: signer,
        address: "wormhole1karc53cm5zyyaeqsw9stmjvu0vwzky7k07lhwm",
        name: "guardianNttAccountant0",
      }),
      client.core.msgCreateAllowlistEntryRequest({
        signer: signer,
        address: "wormhole1cdvy8ae9xgmfjj4pztz77dwqm4wa04glz68r5w",
        name: "guardianNttAccountant1",
      }),
      client.core.msgCreateAllowlistEntryRequest({
        signer: signer,
        address: "wormhole18s5lynnmx37hq4wlrw9gdn68sg2uxp5rwf5k3u",
        name: "nttAccountantTest",
      }),
    ],
    {
      ...ZERO_FEE,
      gas: "10000000",
    }
  );
  console.log(
    "created allowlist entries: ",
    allowListResponse.transactionHash,
    allowListResponse.code
  );

  // instantiate wormhole core bridge
  addresses["cw_wormhole.wasm"] = await instantiate(
    codeIds["cw_wormhole.wasm"],
    {
      gov_chain: govChain,
      gov_address: Buffer.from(govAddress, "hex").toString("base64"),
      guardian_set_expirity: 86400,
      initial_guardian_set: {
        addresses: init_guardians.map((hex) => {
          return {
            bytes: Buffer.from(hex, "hex").toString("base64"),
          };
        }),
        expiration_time: 0,
      },
      chain_id: 3104,
      fee_denom: "utest",
    },
    "wormhole"
  );
  console.log(
    "instantiated wormhole core bridge contract: ",
    addresses["cw_wormhole.wasm"]
  );

  // instantiate wormhole token bridge
  addresses["cw_token_bridge.wasm"] = await instantiate(
    codeIds["cw_token_bridge.wasm"],
    {
      gov_chain: govChain,
      gov_address: Buffer.from(govAddress, "hex").toString("base64"),
      wormhole_contract: addresses["cw_wormhole.wasm"],
      wrapped_asset_code_id: codeIds["cw20_wrapped_2.wasm"],
      chain_id: 3104,
      native_denom: "",
      native_symbol: "",
      native_decimals: 6,
    },
    "tokenBridge"
  );
  console.log(
    "instantiated wormhole token bridge contract: ",
    addresses["cw_token_bridge.wasm"]
  );

  /* Registrations: tell the bridge contracts to know about each other */

  const contract_registrations = {
    "cw_token_bridge.wasm": [
      // Solana
      process.env.REGISTER_SOL_TOKEN_BRIDGE_VAA,
      // Ethereum
      process.env.REGISTER_ETH_TOKEN_BRIDGE_VAA,
      // BSC
      process.env.REGISTER_BSC_TOKEN_BRIDGE_VAA,
      // ALGO
      process.env.REGISTER_ALGO_TOKEN_BRIDGE_VAA,
      // TERRA
      process.env.REGISTER_TERRA_TOKEN_BRIDGE_VAA,
      // TERRA2
      process.env.REGISTER_TERRA2_TOKEN_BRIDGE_VAA,
      // NEAR
      process.env.REGISTER_NEAR_TOKEN_BRIDGE_VAA,
      // APTOS
      process.env.REGISTER_APTOS_TOKEN_BRIDGE_VAA,
    ],
  };

  for (const [contract, registrations] of Object.entries(
    contract_registrations
  )) {
    console.log(`Registering chains for ${contract}:`);
    for (const registration of registrations) {
      const executeMsg = client.wasm.msgExecuteContract({
        sender: signer,
        contract: addresses[contract],
        msg: toUtf8(
          JSON.stringify({
            submit_vaa: {
              data: Buffer.from(registration, "hex").toString("base64"),
            },
          })
        ),
        funds: [],
      });
      const executeRes = await client.signAndBroadcast(signer, [executeMsg], {
        ...ZERO_FEE,
        gas: "10000000",
      });
      console.log(
        "updated token bridge registration: ",
        executeRes.transactionHash
      );
    }
  }

  // add the wasm instantiate allowlist for token bridge
  // contract address bech32 to hex conversion
  const { data } = fromBech32(addresses["cw_token_bridge.wasm"]);
  const contractBuf = Buffer.from(data);

  // code ID number to uint64 hex conversion
  const codeIdBuf = Buffer.alloc(8);
  const cw20CodeId = codeIds["cw20_wrapped_2.wasm"];
  codeIdBuf.writeUInt32BE(cw20CodeId >> 8, 0); //write the high order bits (shifted over)
  codeIdBuf.writeUInt32BE(cw20CodeId & 0x00ff, 4); //write the low order bits
  const payload = `${contractBuf.toString("hex")}${codeIdBuf.toString("hex")}`;
  let vaa: VAA<Other> = {
    version: 1,
    guardianSetIndex: 0,
    signatures: [],
    timestamp: 0,
    nonce: 0,
    emitterChain: GOVERNANCE_CHAIN,
    emitterAddress: GOVERNANCE_EMITTER,
    sequence: BigInt(Math.floor(Math.random() * 100000000)),
    consistencyLevel: 0,
    payload: {
      type: "Other",
      hex: `0000000000000000000000000000000000000000005761736D644D6F64756C65040${CHAIN_ID_WORMCHAIN.toString(
        16
      )}${payload}`,
    },
  };
  vaa.signatures = sign(VAA_SIGNERS, vaa as unknown as VAA<Payload>);

  const msgInstantiateAllowlist = client.core.msgAddWasmInstantiateAllowlist({
    signer: signer,
    address: addresses["cw_token_bridge.wasm"],
    code_id: codeIds["cw20_wrapped_2.wasm"],
    vaa: hexToUint8Array(serialiseVAA(vaa as unknown as VAA<Payload>)),
  });
  const msgInstantiateAllowlistRes = await client.signAndBroadcast(
    signer,
    [msgInstantiateAllowlist],
    {
      ...ZERO_FEE,
      gas: "10000000",
    }
  );
  console.log("wasm instantiate allowlist msg: ", msgInstantiateAllowlist);
  console.log(
    "wasm instantiate allowlist result: ",
    msgInstantiateAllowlistRes
  );

  // instantiate ibc translator
  addresses["ibc_translator.wasm"] = await instantiate(
    codeIds["ibc_translator.wasm"],
    {
      token_bridge_contract: addresses["cw_token_bridge.wasm"],
    },
    "ibcTranslator"
  );
  console.log(
    "instantiated ibc translator contract: ",
    addresses["ibc_translator.wasm"]
  );

  // update channel mapping
  let updateChannelVaa: VAA<Other> = {
    version: 1,
    guardianSetIndex: 0,
    signatures: [],
    timestamp: 0,
    nonce: 0,
    emitterChain: GOVERNANCE_CHAIN,
    emitterAddress: GOVERNANCE_EMITTER,
    sequence: BigInt(Math.floor(Math.random() * 100000000)),
    consistencyLevel: 0,
    payload: {
      type: "Other",
      hex:
        "000000000000000000000000000000000000004962635472616e736c61746f72" + // module IbcTranslator
        "01" + // action IbcReceiverActionUpdateChannelChain
        "0c20" + // target chain id wormchain
        "000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000006368616e6e656c2d31" + // channel-1
        "0012", // chain id terra2 (18)
    },
  };
  updateChannelVaa.signatures = sign(
    VAA_SIGNERS,
    updateChannelVaa as unknown as VAA<Payload>
  );
  const updateMsg = client.wasm.msgExecuteContract({
    sender: signer,
    contract: addresses["ibc_translator.wasm"],
    msg: toUtf8(
      JSON.stringify({
        submit_update_chain_to_channel_map: {
          vaa: Buffer.from(
            serialiseVAA(updateChannelVaa as unknown as VAA<Payload>),
            "hex"
          ).toString("base64"),
        },
      })
    ),
    funds: [],
  });
  const executeRes = await client.signAndBroadcast(signer, [updateMsg], {
    ...ZERO_FEE,
    gas: "10000000",
  });
  console.log("updated channel mapping: ", executeRes.transactionHash);

  // set params for tokenfactory and PFM
  let setDefaultParamsVaa: VAA<Other> = {
    version: 1,
    guardianSetIndex: 0,
    signatures: [],
    timestamp: 0,
    nonce: 0,
    emitterChain: GOVERNANCE_CHAIN,
    emitterAddress: GOVERNANCE_EMITTER,
    sequence: BigInt(Math.floor(Math.random() * 100000000)),
    consistencyLevel: 0,
    payload: {
      type: "Other",
      hex: "",
    },
  };
  const setParamsMsg = client.core.msgExecuteGatewayGovernanceVaa({
    signer: signer,
    vaa: hexToUint8Array(
      serialiseVAA(setDefaultParamsVaa as unknown as VAA<Payload>)
    ),
  });
  await client
    .signAndBroadcast(signer, [setParamsMsg], {
      ...ZERO_FEE,
      gas: "10000000",
    })
    .then((res) => {
      console.log("set params for tokenfactory and pfm: ", res.transactionHash);
    });
}

try {
  main();
} catch (e: any) {
  if (e?.message) {
    console.error(e.message);
  }
  throw e;
}
