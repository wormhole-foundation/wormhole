import yargs from "yargs";
import { config } from "../../config";
import { NETWORK_OPTIONS, RPC_OPTIONS } from "../../consts";
import { NETWORKS } from "../../networks";
import { getProvider, getSigner, publishPackage } from "../../sui";
import { assertNetwork, checkBinary } from "../../utils";
import { YargsAddCommandsFn } from "../Yargs";

export const addDeployCommands: YargsAddCommandsFn = (y: typeof yargs) =>
  y.command(
    "deploy <package-dir>",
    "Deploy a Sui package",
    (yargs) => {
      return yargs
        .positional("package-dir", {
          type: "string",
        })
        .option("network", NETWORK_OPTIONS)
        .option("private-key", {
          alias: "k",
          describe: "Custom private key to sign txs",
          required: false,
          type: "string",
        })
        .option("rpc", RPC_OPTIONS);
    },
    async (argv) => {
      checkBinary("sui", "sui");

      const packageDir = argv["package-dir"];
      const network = argv.network.toUpperCase();
      assertNetwork(network);
      const privateKey = argv["private-key"];
      const rpc = argv.rpc ?? NETWORKS[network].sui.rpc;

      const provider = getProvider(network, rpc);
      const signer = getSigner(provider, network, privateKey);

      console.log("Package", packageDir);
      console.log("RPC", rpc);

      await publishPackage(
        signer,
        network,
        packageDir.startsWith("/") // Allow absolute paths, otherwise assume relative to sui directory
          ? packageDir
          : `${config.wormholeDir}/sui/${packageDir}`
      );
    }
  );
