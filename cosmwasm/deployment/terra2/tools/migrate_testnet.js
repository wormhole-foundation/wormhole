// This script can be used to do a contract migration in testnet. The normal process of submitting a contract upgrade VAA
// does not work because admin of the contracts in testnet still belongs to the deployer wallet.
import { LCDClient, MnemonicKey } from "@terra-money/terra.js";
import { MsgMigrateContract } from "@terra-money/terra.js";
// import axios from "axios";
import yargs from "yargs";
import { hideBin } from "yargs/helpers";

// export const TERRA_GAS_PRICES_URL = "https://fcd.terra.dev/v1/txs/gas_prices";

const argv = yargs(hideBin(process.argv))
  .option("code_id", {
    description: "Which code id to upgrade to",
    type: "number",
  })
  .option("mnemonic", {
    description: "Mnemonic (private key)",
    type: "string",
    required: true,
  })
  .option("contract", {
    description: "Contract address to upgrade",
    type: "string",
    required: true,
  })
  .help()
  .alias("help", "h").argv;

const host = {
  URL: "https://pisco-lcd.terra.dev",
  chainID: "pisco-1",
  name: "testnet",
};

const lcd = new LCDClient(host);

// const feeDenoms = ["uluna"];
// const gasPrices = await axios
//   .get(TERRA_GAS_PRICES_URL)
//   .then((result) => result.data);

const wallet = lcd.wallet(
  new MnemonicKey({
    mnemonic: argv.mnemonic,
  })
);

await wallet.sequence();

/* Do upgrade */

const tx = await wallet.createAndSignTx({
  msgs: [
    new MsgMigrateContract(
      wallet.key.accAddress,
      argv.contract,
      argv.code_id,
      {
        action: "",
      },
      { uluna: 1000 }
    ),
  ],
  memo: "",
  // feeDenoms,
  // gasPrices,
});

const rs = await lcd.tx.broadcast(tx);
console.log(rs);
