import path from "path";
import yargs from "yargs";
import { buildCoin } from "../../chains/sui";
import { NETWORK_OPTIONS } from "../../consts";
import { checkBinary, getNetwork } from "../../utils";
import { YargsAddCommandsFn } from "../Yargs";

const README_URL =
  "https://github.com/wormhole-foundation/wormhole/blob/main/sui/README.md";

export const addBuildCommands: YargsAddCommandsFn = (y: typeof yargs) =>
  y.command(
    "build-coin",
    `Build wrapped coin and dump bytecode.

    Example:
      worm sui build-coin -d 8 -v V__0_1_1 -n testnet`,
    (yargs) =>
      yargs
        .option("decimals", {
          alias: "d",
          describe: "Decimals of asset",
          demandOption: true,
          type: "number",
        })
        // Can't be called version because of a conflict with the native version option
        .option("version-struct", {
          alias: "v",
          describe: "Version control struct name (e.g. V__0_1_0)",
          demandOption: true,
          type: "string",
        })
        .option("network", NETWORK_OPTIONS)
        .option("package-path", {
          alias: "p",
          describe: "Path to coin module",
          demandOption: false,
          type: "string",
        }),
    async (argv) => {
      checkBinary("sui", README_URL);

      const network = getNetwork(argv.network);
      const decimals = argv["decimals"];
      const version = argv["version-struct"];
      const packagePath =
        argv["package-path"] ??
        path.resolve(__dirname, "../../../../../sui/examples");

      // Note: In Sui v1.63+, dependencies are resolved automatically via
      // Pub.localnet.toml (for ephemeral) or Published.toml (for persistent).
      const build = await buildCoin(network, packagePath, version, decimals);
      console.log(build);
      console.log(
        "Bytecode hex:",
        Buffer.from(build.modules[0], "base64").toString("hex")
      );
    }
  );
