import "dotenv/config";
import { LCDClient, MnemonicKey } from "@terra-money/terra.js";
import {
  Fee,
  MsgInstantiateContract,
  MsgExecuteContract,
  MsgStoreCode,
} from "@terra-money/terra.js";
import { readFileSync, readdirSync } from "fs";
import { Bech32, toHex } from "@cosmjs/encoding";
import { zeroPad } from "ethers/lib/utils.js";

function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

async function broadcastAndWait(terra, tx) {
  const response = await terra.tx.broadcast(tx);
  let currentHeight = (await terra.tendermint.blockInfo()).block.header.height;
  while (currentHeight <= response.height) {
    await sleep(100);
    currentHeight = (await terra.tendermint.blockInfo()).block.header.height;
  }
  return response;
}

/*
  NOTE: Only append to this array: keeping the ordering is crucial, as the
  contracts must be imported in a deterministic order so their addresses remain
  deterministic.
*/
const artifacts = [
  "wormhole.wasm",
  "token_bridge_terra.wasm",
  "cw20_wrapped.wasm",
  "cw20_base.wasm",
  "mock_bridge_integration.wasm",
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

const unexpected_artifacts = actual_artifacts.filter(
  (a) => !artifacts.includes(a)
);
if (unexpected_artifacts.length) {
  console.log(
    "Error during terra deployment. The following files are not expected to be in the artifacts folder:"
  );
  unexpected_artifacts.forEach((file) => console.log(`  - ${file}`));
  console.log("Hint: you might need to modify tools/deploy.js");
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

    const rs = await broadcastAndWait(terra, tx);
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
    .then((tx) => broadcastAndWait(terra, tx))
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

addresses["wormhole.wasm"] = await instantiate(
  "wormhole.wasm",
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
  },
  "wormhole"
);

addresses["token_bridge_terra.wasm"] = await instantiate(
  "token_bridge_terra.wasm",
  {
    gov_chain: govChain,
    gov_address: Buffer.from(govAddress, "hex").toString("base64"),
    wormhole_contract: addresses["wormhole.wasm"],
    wrapped_asset_code_id: codeIds["cw20_wrapped.wasm"],
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
      {
        address: "terra17tv2hvwpg0ukqgd2y5ct2w54fyan7z0zxrm2f9",
        amount: "100000000",
      },
    ],
    mint: null,
  },
  "mock"
);

/* Registrations: tell the bridge contracts to know about each other */

const contract_registrations = {
  "token_bridge_terra.wasm": [
    // Solana
    process.env.REGISTER_SOL_TOKEN_BRIDGE_VAA,
    // Ethereum
    process.env.REGISTER_ETH_TOKEN_BRIDGE_VAA,
    // BSC
    process.env.REGISTER_BSC_TOKEN_BRIDGE_VAA,
    // ALGO
    process.env.REGISTER_ALGO_TOKEN_BRIDGE_VAA,
    // TERRA2
    process.env.REGISTER_TERRA2_TOKEN_BRIDGE_VAA,
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
        fee: new Fee(2000000, {
          uluna: "100000",
        }),
      })
      .then((tx) => broadcastAndWait(terra, tx))
      .then((rs) => console.log(rs));
  }
}

// Terra addresses are "human-readable", but for cross-chain registrations, we
// want the "canonical" version
function convert_terra_address_to_hex(human_addr) {
  return "0x" + toHex(zeroPad(Bech32.decode(human_addr).data, 32));
}
