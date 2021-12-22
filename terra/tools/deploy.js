import { Wallet, LCDClient, MnemonicKey } from "@terra-money/terra.js";
import {
  StdFee,
  MsgInstantiateContract,
  MsgExecuteContract,
  MsgStoreCode,
} from "@terra-money/terra.js";
import { readFileSync, readdirSync } from "fs";

// TODO: Workaround /tx/estimate_fee errors.

const gas_prices = {
  uluna: "0.15",
  usdr: "0.1018",
  uusd: "0.15",
  ukrw: "178.05",
  umnt: "431.6259",
  ueur: "0.125",
  ucny: "0.97",
  ujpy: "16",
  ugbp: "0.11",
  uinr: "11",
  ucad: "0.19",
  uchf: "0.13",
  uaud: "0.19",
  usgd: "0.2",
};

async function main() {
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

  // Deploy WASM blobs.
  // Read a list of files from directory containing compiled contracts.
  const artifacts = readdirSync("../artifacts/");

  // Sort them to get a determinstic list for consecutive code ids.
  artifacts.sort();
  artifacts.reverse();

  const hardcodedGas = {
    "cw20_base.wasm": 4000000,
    "cw20_wrapped.wasm": 4000000,
    "wormhole.wasm": 5000000,
    "token_bridge.wasm": 6000000,
    "pyth_bridge.wasm": 5000000,
  };

  // Deploy all found WASM files and assign Code IDs.
  const codeIds = {};
  for (const artifact in artifacts) {
    if (
      artifacts.hasOwnProperty(artifact) &&
      artifacts[artifact].includes(".wasm")
    ) {
      const file = artifacts[artifact];
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
          fee: new StdFee(hardcodedGas[artifacts[artifact]], {
            uluna: "100000",
          }),
        });

        const rs = await terra.tx.broadcast(tx);
        const ci = /"code_id","value":"([^"]+)/gm.exec(rs.raw_log)[1];
        codeIds[file] = parseInt(ci);
      } catch (e) {
        console.log("Failed to Execute");
      }
    }
  }

  console.log(codeIds);

  // Governance constants defined by the Wormhole spec.
  const govChain = 1;
  const govAddress =
    "0000000000000000000000000000000000000000000000000000000000000004";
  const addresses = {};

  // Instantiate Wormhole
  console.log("Instantiating Wormhole");
  await wallet
    .createAndSignTx({
      msgs: [
        new MsgInstantiateContract(
          wallet.key.accAddress,
          wallet.key.accAddress,
          codeIds["wormhole.wasm"],
          {
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
          }
        ),
      ],
      memo: "",
    })
    .then((tx) => terra.tx.broadcast(tx))
    .then((rs) => {
      const address = /"contract_address","value":"([^"]+)/gm.exec(
        rs.raw_log
      )[1];
      addresses["wormhole.wasm"] = address;
    });

  console.log("Instantiating Token Bridge");
  await wallet
    .createAndSignTx({
      msgs: [
        new MsgInstantiateContract(
          wallet.key.accAddress,
          wallet.key.accAddress,
          codeIds["token_bridge.wasm"],
          {
            owner: wallet.key.accAddress,
            gov_chain: govChain,
            gov_address: Buffer.from(govAddress, "hex").toString("base64"),
            wormhole_contract: addresses["wormhole.wasm"],
            wrapped_asset_code_id: codeIds["cw20_wrapped.wasm"],
          }
        ),
      ],
      memo: "",
    })
    .then((tx) => terra.tx.broadcast(tx))
    .then((rs) => {
      const address = /"contract_address","value":"([^"]+)/gm.exec(
        rs.raw_log
      )[1];
      addresses["token_bridge.wasm"] = address;
    });

  await wallet
    .createAndSignTx({
      msgs: [
        new MsgInstantiateContract(
          wallet.key.accAddress,
          undefined,
          codeIds["cw20_base.wasm"],
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
          }
        ),
      ],
      memo: "",
    })
    .then((tx) => terra.tx.broadcast(tx))
    .then((rs) => {
      const address = /"contract_address","value":"([^"]+)/gm.exec(
        rs.raw_log
      )[1];
      addresses["mock.wasm"] = address;
    });

  const pythEmitterAddress =
    "71f8dcb863d176e2c420ad6610cf687359612b6fb392e0642b0ca6b1f186aa3b";
  const pythChain = 1;

  // Instantiate Pyth over Wormhole
  console.log("Instantiating Pyth over Wormhole");
  await wallet
    .createAndSignTx({
      msgs: [
        new MsgInstantiateContract(
          wallet.key.accAddress,
          wallet.key.accAddress,
          codeIds["pyth_bridge.wasm"],
          {
            gov_chain: govChain,
            gov_address: Buffer.from(govAddress, "hex").toString("base64"),
            wormhole_contract: addresses["wormhole.wasm"],
            pyth_emitter: Buffer.from(pythEmitterAddress, "hex").toString(
              "base64"
            ),
            pyth_emitter_chain: pythChain,
          }
        ),
      ],
      memo: "",
    })
    .then((tx) => terra.tx.broadcast(tx))
    .then((rs) => {
      const address = /"contract_address","value":"([^"]+)/gm.exec(
        rs.raw_log
      )[1];
      addresses["pyth_bridge.wasm"] = address;
    });

  console.log(addresses);

  const registrations = [
    "01000000000100c9f4230109e378f7efc0605fb40f0e1869f2d82fda5b1dfad8a5a2dafee85e033d155c18641165a77a2db6a7afbf2745b458616cb59347e89ae0c7aa3e7cc2d400000000010000000100010000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000546f6b656e4272696467650100000001c69a1b1a65dd336bf1df6a77afb501fc25db7fc0938cb08595a9ef473265cb4f",
    "01000000000100e2e1975d14734206e7a23d90db48a6b5b6696df72675443293c6057dcb936bf224b5df67d32967adeb220d4fe3cb28be515be5608c74aab6adb31099a478db5c01000000010000000100010000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000546f6b656e42726964676501000000020000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16",
  ];

  for (const registration in registrations) {
    if (registrations.hasOwnProperty(registration)) {
      console.log('Registering');
      await wallet
        .createAndSignTx({
          msgs: [
            new MsgExecuteContract(
              wallet.key.accAddress,
              addresses["token_bridge.wasm"],
              {
                submit_vaa: {
                  data: Buffer.from(registrations[registration], "hex").toString('base64'),
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
}

main();
