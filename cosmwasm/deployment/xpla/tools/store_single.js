import "dotenv/config";
import { LCDClient, MnemonicKey, MsgStoreCode } from "@xpla/xpla.js";
import { readFileSync } from "fs";
import yargs from "yargs";
import { hideBin } from "yargs/helpers";

const argv = yargs(hideBin(process.argv))
  .option("network", {
    description: "Which network to deploy to",
    choices: ["mainnet", "testnet", "devnet"],
    required: true,
  })
  .option("artifact", {
    description: "Which WASM file to deploy",
    type: "string",
    required: true,
  })
  .option("mnemonic", {
    description: "Mnemonic (private key)",
    type: "string",
    required: true,
  })
  .help()
  .alias("help", "h").argv;

const artifact = argv.artifact;

/* Set up terra client & wallet */

const host =
  argv.network === "mainnet"
    ? {
        URL: "https://dimension-lcd.xpla.dev",
        chainID: "dimension_37-1",
        name: "mainnet",
      }
    : argv.network === "testnet"
    ? {
        URL: "https://cube-lcd.xpla.dev:443",
        chainID: "cube_47-5",
        name: "testnet",
      }
    : {
        URL: undefined,
        chainID: undefined,
      };

const lcd = new LCDClient({
  URL: host.URL,
  chainID: host.chainID,
});

const wallet = lcd.wallet(
  new MnemonicKey({
    mnemonic: argv.mnemonic,
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

const tx = await wallet.createAndSignTx({
  msgs: [store_code],
  memo: "",
});

const rs = await lcd.tx.broadcast(tx);
const ci = /"code_id","value":"([^"]+)/gm.exec(rs.raw_log)[1];
codeId = parseInt(ci);

console.log("Code ID: ", codeId);
