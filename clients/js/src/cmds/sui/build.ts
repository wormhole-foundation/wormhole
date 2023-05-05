import path from "path";
import yargs from "yargs";
import { CONTRACTS, NETWORK_OPTIONS, RPC_OPTIONS } from "../../consts";
import { NETWORKS } from "../../networks";
import { buildCoin, getProvider } from "../../sui";
import { assertNetwork, checkBinary } from "../../utils";
import { YargsAddCommandsFn } from "../Yargs";

const README_URL =
  "https://github.com/wormhole-foundation/wormhole/blob/main/sui/README.md";

export const addBuildCommands: YargsAddCommandsFn = (y: typeof yargs) =>
  y.command(
    "build-coin",
    `Build wrapped coin and dump bytecode.
    
    Example:
      worm sui build-coin -d 8 -v V__0_1_1 -n testnet -r "https://fullnode.testnet.sui.io:443"`,
    (yargs) =>
      yargs
        .option("decimals", {
          alias: "d",
          describe: "Decimals of asset",
          required: true,
          type: "number",
        })
        // Can't be called version because of a conflict with the native version option
        .option("version-struct", {
          alias: "v",
          describe: "Version control struct name (e.g. V__0_1_0)",
          required: true,
          type: "string",
        })
        .option("network", NETWORK_OPTIONS)
        .option("package-path", {
          alias: "p",
          describe: "Path to coin module",
          required: false,
          type: "string",
        })
        .option("wormhole-state", {
          alias: "w",
          describe: "Wormhole state object ID",
          required: false,
          type: "string",
        })
        .option("token-bridge-state", {
          alias: "t",
          describe: "Token bridge state object ID",
          required: false,
          type: "string",
        })
        .option("rpc", RPC_OPTIONS),
    async (argv) => {
      checkBinary("sui", README_URL);

      const network = argv.network.toUpperCase();
      assertNetwork(network);
      const decimals = argv["decimals"];
      const version = argv["version-struct"];
      const packagePath =
        argv["package-path"] ??
        path.resolve(__dirname, "../../../../../sui/examples");
      const coreBridgeStateObjectId =
        argv["wormhole-state"] ?? CONTRACTS[network].sui.core;
      const tokenBridgeStateObjectId =
        argv["token-bridge-state"] ?? CONTRACTS[network].sui.token_bridge;
      const provider = getProvider(
        network,
        argv.rpc ?? NETWORKS[network].sui.rpc
      );

      const build = await buildCoin(
        provider,
        network,
        packagePath,
        coreBridgeStateObjectId,
        tokenBridgeStateObjectId,
        version,
        decimals
      );
      console.log(build);
      console.log(
        "Bytecode hex:",
        Buffer.from(build.modules[0], "base64").toString("hex")
      );
    }
  );
