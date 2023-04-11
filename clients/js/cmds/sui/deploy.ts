import yargs from "yargs";
import { NETWORK_OPTIONS, RPC_OPTIONS } from "../../consts";
import { NETWORKS } from "../../networks";
import { getProvider, getSigner, publishPackage } from "../../sui";
import {
  logCreatedObjects,
  logPublishedPackageId,
  logTransactionDigest,
  logTransactionSender,
} from "../../sui/log";
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

      // Allow absolute paths, otherwise assume relative to directory `worm` command is run from
      const dir = packageDir.startsWith("/")
        ? packageDir
        : `${process.cwd()}/${packageDir}`;
      const packagePath = dir.endsWith("/") ? dir.slice(0, -1) : dir;
      const res = await publishPackage(signer, network, packagePath);

      // Dump deployment info to console
      logTransactionDigest(res);
      logTransactionSender(res);
      logPublishedPackageId(res);
      logCreatedObjects(res);
    }
  );
