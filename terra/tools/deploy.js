import { LCDClient, MnemonicKey } from "@terra-money/terra.js";
import {
  StdFee,
  MsgInstantiateContract,
  MsgExecuteContract,
  MsgStoreCode,
} from "@terra-money/terra.js";
import { readFileSync, readdirSync } from "fs";
import { Bech32, toHex } from "@cosmjs/encoding";
import { zeroPad } from "ethers/lib/utils.js";

/*
  NOTE: Only append to this array: keeping the ordering is crucial, as the
  contracts must be imported in a deterministic order so their addresses remain
  deterministic.
*/
const artifacts = [
  "wormhole.wasm",
  "token_bridge.wasm",
  "cw20_wrapped.wasm",
  "cw20_base.wasm",
  "nft_bridge.wasm",
  "cw721_wrapped.wasm",
  "cw721_base.wasm",
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

async function instantiate(contract, inst_msg) {
  var address;
  await wallet
    .createAndSignTx({
      msgs: [
        new MsgInstantiateContract(
          wallet.key.accAddress,
          wallet.key.accAddress,
          codeIds[contract],
          inst_msg
        ),
      ],
      memo: "",
    })
    .then((tx) => terra.tx.broadcast(tx))
    .then((rs) => {
      address = /"contract_address","value":"([^"]+)/gm.exec(rs.raw_log)[1];
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

addresses["wormhole.wasm"] = await instantiate("wormhole.wasm", {
  gov_chain: govChain,
  gov_address: Buffer.from(govAddress, "hex").toString("base64"),
  guardian_set_expirity: 86400,
  initial_guardian_set: {
    addresses: [
      {
        bytes: Buffer.from(
          "beFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe",
          "hex"
        ).toString("base64"),
      },
    ],
    expiration_time: 0,
  },
});

addresses["token_bridge.wasm"] = await instantiate("token_bridge.wasm", {
  gov_chain: govChain,
  gov_address: Buffer.from(govAddress, "hex").toString("base64"),
  wormhole_contract: addresses["wormhole.wasm"],
  wrapped_asset_code_id: codeIds["cw20_wrapped.wasm"],
});

addresses["mock.wasm"] = await instantiate("cw20_base.wasm", {
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
});

addresses["nft_bridge.wasm"] = await instantiate("nft_bridge.wasm", {
  gov_chain: govChain,
  gov_address: Buffer.from(govAddress, "hex").toString("base64"),
  wormhole_contract: addresses["wormhole.wasm"],
  wrapped_asset_code_id: codeIds["cw721_wrapped.wasm"],
});

addresses["cw721_base.wasm"] = await instantiate("cw721_base.wasm", {
  name: "MOCK",
  symbol: "MCK",
  minter: wallet.key.accAddress,
});

async function mint_cw721(token_id, token_uri) {
  await wallet
    .createAndSignTx({
      msgs: [
        new MsgExecuteContract(
          wallet.key.accAddress,
          addresses["cw721_base.wasm"],
          {
            mint: {
              token_id: token_id.toString(),
              owner: wallet.key.accAddress,
              token_uri: token_uri,
            },
          },
          { uluna: 1000 }
        ),
      ],
      memo: "",
      fee: new StdFee(2000000, {
        uluna: "100000",
      }),
    })
    .then((tx) => terra.tx.broadcast(tx));
  console.log(
    `Minted NFT with token_id ${token_id} at ${addresses["cw721_base.wasm"]}`
  );
}

await mint_cw721(
  0,
  "https://ixmfkhnh2o4keek2457f2v2iw47cugsx23eynlcfpstxihsay7nq.arweave.net/RdhVHafTuKIRWud-XVdItz4qGlfWyYasRXyndB5Ax9s/"
);
await mint_cw721(
  1,
  "https://portal.neondistrict.io/api/getNft/158456327500392944014123206890"
);

/* Registrations: tell the bridge contracts to know about each other */

const contract_registrations = {
  "token_bridge.wasm": [
    // Solana
    "01000000000100c9f4230109e378f7efc0605fb40f0e1869f2d82fda5b1dfad8a5a2dafee85e033d155c18641165a77a2db6a7afbf2745b458616cb59347e89ae0c7aa3e7cc2d400000000010000000100010000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000546f6b656e4272696467650100000001c69a1b1a65dd336bf1df6a77afb501fc25db7fc0938cb08595a9ef473265cb4f",
    // Ethereum
    "01000000000100e2e1975d14734206e7a23d90db48a6b5b6696df72675443293c6057dcb936bf224b5df67d32967adeb220d4fe3cb28be515be5608c74aab6adb31099a478db5c01000000010000000100010000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000546f6b656e42726964676501000000020000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16",
    // BSC
    "01000000000100719b4ada436f614489dbf87593c38ba9aea35aa7b997387f8ae09f819806f5654c8d45b6b751faa0e809ccbc294794885efa205bd8a046669464c7cbfb03d183010000000100000001000100000000000000000000000000000000000000000000000000000000000000040000000002c8bb0600000000000000000000000000000000000000000000546f6b656e42726964676501000000040000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16",
  ],
  "nft_bridge.wasm": [
    // Solana
    "010000000001007985ba742002ae745c19722fea4d82102e68526c7c9d94d0e5d0a809071c98451c9693b230b3390f4ca9555a3ba9a9abbe87cf6f9e400682213e4fbbe1dabb9e0100000001000000010001000000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000004e4654427269646765010000000196ee982293251b48729804c8e8b24b553eb6b887867024948d2236fd37a577ab",
    // Ethereum
    "01000000000100d073f81a4ecf2469b0674b1902dcbcad2da7f70ecdd7e1aec65414380ca2c05426380c33bb083ab41167c522231c1485b9c8ffce04eaf2e8a6f50edaa72074c50000000001000000010001000000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000004e4654427269646765010000000200000000000000000000000026b4afb60d6c903165150c6f0aa14f8016be4aec",
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
        fee: new StdFee(2000000, {
          uluna: "100000",
        }),
      })
      .then((tx) => terra.tx.broadcast(tx))
      .then((rs) => console.log(rs));
  }
}

// Terra addresses are "human-readable", but for cross-chain registrations, we
// want the "canonical" version
function convert_terra_address_to_hex(human_addr) {
  return "0x" + toHex(zeroPad(Bech32.decode(human_addr).data, 32));
}
