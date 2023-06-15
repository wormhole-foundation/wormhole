import "dotenv/config";
import { LCDClient, MnemonicKey } from "@terra-money/terra.js";
import {
  MsgInstantiateContract,
  MsgExecuteContract,
  MsgStoreCode,
} from "@terra-money/terra.js";
import { readFileSync, readdirSync } from "fs";
import { Bech32, toHex } from "@cosmjs/encoding";
import { zeroPad } from "ethers/lib/utils.js";

// Generated using
// `guardiand template ibc-receiver-update-channel-chain --channel-id channel-0 --chain-id 3104 --target-chain-id 32 > terra2.prototxt`
// `guardiand admin governance-vaa-verify terra2.prototxt`
const WORMHOLE_IBC_WHITELIST_VAA =
  "0100000000010025e55ab23c8d0a7fddd4686f41801792cdce1ff7335a2b9436192bd552fa0f9b5c18016057b0d4b3f24c759eafe3e5fedd7fce76fe6f21cec815ffbaf4ec3ad801000000009b9a6b2d0001000000000000000000000000000000000000000000000000000000000000000460efd4405060ac0c200000000000000000000000000000000000000000004962635265636569766572010020000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000006368616e6e656c2d300c20";

/*
  NOTE: Only append to this array: keeping the ordering is crucial, as the
  contracts must be imported in a deterministic order so their addresses remain
  deterministic.
*/
const artifacts = [
  "cw_wormhole.wasm",
  "cw_token_bridge.wasm",
  "cw20_wrapped_2.wasm",
  "cw20_base.wasm",
  "wormhole_ibc.wasm",
];

/* Check that the artifact folder contains all the wasm files we expect and nothing else */

const actual_artifacts = readdirSync("../artifacts/").filter((a) =>
  a.endsWith(".wasm")
);

const missing_artifacts = artifacts.filter(
  (a) => !actual_artifacts.includes(a)
);
if (missing_artifacts.length) {
  console.log(
    "Error during terra deployment. The following files are expected to be in the artifacts folder:"
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

/* Set up terra client & wallet */

const terra = new LCDClient({
  URL: "http://localhost:1317",
  chainID: "localterra",
});

const wallet = terra.wallet(
  new MnemonicKey({
    mnemonic:
      "notice oak worry limit wrap speak medal online prefer cluster roof addict wrist behave treat actual wasp year salad speed social layer crew genius",
  })
);

await wallet.sequence();

/* Deploy artifacts */

const codeIds = {};
for (const file of artifacts) {
  const contract_bytes = readFileSync(`../artifacts/${file}`);
  console.log(`Storing WASM: ${file} (${contract_bytes.length} bytes)`);

  const store_code = new MsgStoreCode(
    wallet.key.accAddress,
    contract_bytes.toString("base64")
  );

  try {
    const tx = await wallet.createAndSignTx({
      msgs: [store_code],
      memo: "",
    });

    const rs = await terra.tx.broadcast(tx);
    const ci = /"code_id","value":"([^"]+)/gm.exec(rs.raw_log)[1];
    codeIds[file] = parseInt(ci);
  } catch (e) {
    console.log(`${e}`);
  }
}

console.log(codeIds);

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

async function instantiate(contract, inst_msg, label) {
  var address;
  await wallet
    .createAndSignTx({
      msgs: [
        new MsgInstantiateContract(
          wallet.key.accAddress,
          wallet.key.accAddress,
          codeIds[contract],
          inst_msg,
          undefined,
          label
        ),
      ],
      memo: "",
    })
    .then((tx) => terra.tx.broadcast(tx))
    .then((rs) => {
      address = /"_contract_address","value":"([^"]+)/gm.exec(rs.raw_log)[1];
    });
  console.log(
    `Instantiated ${contract} at ${address} (${convert_terra_address_to_hex(
      address
    )})`
  );
  return address;
}

// Instantiate contracts.  NOTE: Only append at the end, the ordering must be
// deterministic for the addresses to work

const addresses = {};

const init_guardians = JSON.parse(process.env.INIT_SIGNERS);
if (!init_guardians || init_guardians.length === 0) {
  throw "failed to get initial guardians from .env file.";
}

addresses["cw_wormhole.wasm"] = await instantiate(
  "cw_wormhole.wasm",
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
    chain_id: 18,
    fee_denom: "uluna",
  },
  "wormhole"
);

addresses["cw_token_bridge.wasm"] = await instantiate(
  "cw_token_bridge.wasm",
  {
    gov_chain: govChain,
    gov_address: Buffer.from(govAddress, "hex").toString("base64"),
    wormhole_contract: addresses["cw_wormhole.wasm"],
    wrapped_asset_code_id: codeIds["cw20_wrapped_2.wasm"],
    chain_id: 18,
    native_denom: "uluna",
    native_symbol: "LUNA",
    native_decimals: 6,
  },
  "tokenBridge"
);

addresses["mock.wasm"] = await instantiate(
  "cw20_base.wasm",
  {
    name: "MOCK",
    symbol: "MCK",
    decimals: 6,
    initial_balances: [
      {
        address: wallet.key.accAddress,
        amount: "100000000",
      },
    ],
    mint: null,
  },
  "mock"
);

addresses["wormhole_ibc.wasm"] = await instantiate(
  "wormhole_ibc.wasm",
  {
    gov_chain: govChain,
    gov_address: Buffer.from(govAddress, "hex").toString("base64"),
    guardian_set_expirity: 86400,
    initial_guardian_set: {
      // This is using one guardian so the above registration can be hard-coded
      // TODO: instantiate with the correct guardian set and dynamically generate the registration
      addresses: [
        {
          bytes: Buffer.from(init_guardians[0], "hex").toString("base64"),
        },
      ],
      expiration_time: 0,
    },
    chain_id: 32,
    fee_denom: "uluna",
  },
  "wormholeIbc"
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
    // NEAR
    process.env.REGISTER_NEAR_TOKEN_BRIDGE_VAA,
    // Wormhole Chain
    process.env.REGISTER_WORMCHAIN_TOKEN_BRIDGE_VAA,
    // APTOS
    process.env.REGISTER_APTOS_TOKEN_BRIDGE_VAA,
  ],
};

for (const [contract, registrations] of Object.entries(
  contract_registrations
)) {
  console.log(`Registering chains for ${contract}:`);
  for (const registration of registrations) {
    await wallet
      .createAndSignTx({
        msgs: [
          new MsgExecuteContract(
            wallet.key.accAddress,
            addresses[contract],
            {
              submit_vaa: {
                data: Buffer.from(registration, "hex").toString("base64"),
              },
            },
            { uluna: 1000 }
          ),
        ],
        memo: "",
      })
      .then((tx) => terra.tx.broadcast(tx))
      .then((rs) => console.log(rs))
      .catch((error) => {
        if (error.response) {
          // Request made and server responded
          console.error(
            error.response.data,
            error.response.status,
            error.response.headers
          );
        } else if (error.request) {
          // The request was made but no response was received
          console.error(error.request);
        } else {
          // Something happened in setting up the request that triggered an Error
          console.error("Error", error.message);
        }

        throw new Error(`Registering chain failed: ${registration}`);
      });
  }
}

// submit wormchain channel ID whitelist to the wormhole_ibc contract
const ibc_whitelist_tx = await wallet.createAndSignTx({
  msgs: [
    new MsgExecuteContract(
      wallet.key.accAddress,
      addresses["wormhole_ibc.wasm"],
      {
        submit_update_channel_chain: {
          vaa: Buffer.from(WORMHOLE_IBC_WHITELIST_VAA, "hex").toString(
            "base64"
          ),
        },
      },
      { uluna: 1000 }
    ),
  ],
  memo: "",
});
const ibc_whitelist_res = await terra.tx.broadcast(ibc_whitelist_tx);
console.log("updated wormhole_ibc channel whitelist", ibc_whitelist_res.txhash);

// Terra addresses are "human-readable", but for cross-chain registrations, we
// want the "canonical" version
function convert_terra_address_to_hex(human_addr) {
  return "0x" + toHex(zeroPad(Bech32.decode(human_addr).data, 32));
}
