import { LCDClient, MnemonicKey } from "@terra-money/terra.js";
import { MsgStoreCode } from "@terra-money/terra.js";
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
        URL: "https://phoenix-lcd.terra.dev",
        chainID: "phoenix-1",
        name: "mainnet",
      }
    : argv.network === "testnet"
    ? {
        URL: "https://pisco-lcd.terra.dev",
        chainID: "pisco-1",
        name: "testnet",
      }
    : {
        URL: "http://localhost:1318",
        chainID: "phoenix-1",
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
