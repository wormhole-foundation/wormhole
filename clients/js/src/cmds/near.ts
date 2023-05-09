import yargs from "yargs";
import { deploy_near, upgrade_near } from "../near";

// Near utilities
export const command = "near";
export const desc = "NEAR utilities";
export const builder = function (y: typeof yargs) {
  return y
    .option("module", {
      alias: "m",
      describe: "Module to query",
      type: "string",
      choices: ["Core", "NFTBridge", "TokenBridge"],
      required: false,
    })
    .option("network", {
      alias: "n",
      describe: "network",
      type: "string",
      choices: ["mainnet", "testnet", "devnet"],
      required: true,
    })
    .option("account", {
      describe: "near deployment account",
      type: "string",
      required: true,
    })
    .option("attach", {
      describe: "attach some near",
      type: "string",
      required: false,
    })
    .option("target", {
      describe: "near account to upgrade",
      type: "string",
      required: false,
    })
    .option("mnemonic", {
      describe: "near private keys",
      type: "string",
      required: false,
    })
    .option("keys", {
      describe: "near private keys",
      type: "string",
      required: false,
    })
    .command(
      "contract-update <file>",
      "Submit a contract update using our specific APIs",
      (yargs) => {
        return yargs.positional("file", {
          type: "string",
          describe: "wasm",
        });
      },
      async (argv) => {
        await upgrade_near(argv);
      }
    )
    .command(
      "deploy <file>",
      "Submit a contract update using near APIs",
      (yargs) => {
        return yargs.positional("file", {
          type: "string",
          describe: "wasm",
        });
      },
      async (argv) => {
        await deploy_near(argv);
      }
    );
};

export const handler = (argv) => {};
