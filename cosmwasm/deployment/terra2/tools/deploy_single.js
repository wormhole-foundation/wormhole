import { LCDClient, MnemonicKey } from "@terra-money/terra.js";
import {
  MsgInstantiateContract,
  MsgStoreCode,
} from "@terra-money/terra.js";
import { readFileSync } from "fs";
import { Bech32, toHex } from "@cosmjs/encoding";
import { zeroPad } from "ethers/lib/utils.js";
import axios from "axios";
import yargs from "yargs";
import {hideBin} from "yargs/helpers";

export const TERRA_GAS_PRICES_URL = "https://fcd.terra.dev/v1/txs/gas_prices";

const argv = yargs(hideBin(process.argv))
  .option('network', {
    description: 'Which network to deploy to',
    choices: ['mainnet', 'testnet', 'devnet'],
    required: true
  })
  .option('artifact', {
    description: 'Which WASM file to deploy',
    type: 'string',
    required: true
  })
  .option('mnemonic', {
    description: 'Mnemonic (private key)',
    type: 'string',
    required: true
  })
  .help()
  .alias('help', 'h').argv;

const artifact = argv.artifact;

/* Set up terra client & wallet */

const terra_host =
      argv.network === "mainnet"
    ? {
        URL: "https://lcd.terra.dev",
        chainID: "columbus-5",
        name: "mainnet",
      }
    : argv.network === "testnet"
    ? {
        URL: "https://bombay-lcd.terra.dev",
        chainID: "bombay-12",
        name: "testnet",
      }
    : {
        URL: "http://localhost:1317",
        chainID: "localterra",
      };

const lcd = new LCDClient(terra_host);

const feeDenoms = ["uluna"];

const gasPrices = await axios
  .get(TERRA_GAS_PRICES_URL)
  .then((result) => result.data);

const wallet = lcd.wallet(
  new MnemonicKey({
    mnemonic: argv.mnemonic
  })
);

await wallet.sequence();

/* Deploy artifacts */

let codeId;
const contract_bytes = readFileSync(artifact);
console.log(`Storing WASM: ${artifact} (${contract_bytes.length} bytes)`);

const store_code = new MsgStoreCode(
  wallet.key.accAddress,
  contract_bytes.toString("base64")
);

const feeEstimate = await lcd.tx.estimateFee(
  wallet.key.accAddress,
  [store_code],
  {
    memo: "",
    feeDenoms,
    gasPrices,
  }
);

console.log("Deploy fee: ", feeEstimate.amount.toString());

const tx = await wallet.createAndSignTx({
  msgs: [store_code],
  memo: "",
  feeDenoms,
  gasPrices,
  fee: feeEstimate,
});

const rs = await lcd.tx.broadcast(tx);
const ci = /"code_id","value":"([^"]+)/gm.exec(rs.raw_log)[1];
codeId = parseInt(ci);

console.log("Code ID: ", codeId);

/* Instantiate contracts.
 *
 * We instantiate the core contracts here (i.e. wormhole itself and the bridge contracts).
 * The wrapped asset contracts don't need to be instantiated here, because those
 * will be instantiated by the on-chain bridge contracts on demand.
 * */
async function instantiate(codeId, inst_msg) {
  var address;
  await wallet
    .createAndSignTx({
      msgs: [
        new MsgInstantiateContract(
          wallet.key.accAddress,
          wallet.key.accAddress,
          codeId,
          inst_msg
        ),
      ],
      memo: "",
    })
    .then((tx) => lcd.tx.broadcast(tx))
    .then((rs) => {
      address = /"contract_address","value":"([^"]+)/gm.exec(rs.raw_log)[1];
    });
  console.log(`Instantiated ${contract} at ${address} (${convert_terra_address_to_hex(address)})`);
  return address;
}

// example usage of instantiate:

// const contractAddress = await instantiate("wormhole.wasm", {
//   gov_chain: govChain,
//   gov_address: Buffer.from(govAddress, "hex").toString("base64"),
//   guardian_set_expirity: 86400,
//   initial_guardian_set: {
//     addresses: [
//       {
//         bytes: Buffer.from(
//           "beFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe",
//           "hex"
//         ).toString("base64"),
//       },
//     ],
//     expiration_time: 0,
//   },
//   chain_id: 18,
//   fee_denom: "uluna",
// });


// Terra addresses are "human-readable", but for cross-chain registrations, we
// want the "canonical" version
function convert_terra_address_to_hex(human_addr) {
  return "0x" + toHex(zeroPad(Bech32.decode(human_addr).data, 32));
}
