import { SuiTransactionBlockResponse } from "@mysten/sui.js";
import fs from "fs";
import yargs from "yargs";
import {
  DEBUG_OPTIONS,
  NETWORK_OPTIONS,
  PRIVATE_KEY_OPTIONS,
  RPC_OPTIONS,
} from "../../consts";
import { NETWORKS } from "../../networks";
import { getProvider, getSigner, publishPackage } from "../../sui";
import {
  logCreatedObjects,
  logPublishedPackageId,
  logTransactionDigest,
  logTransactionSender,
} from "../../sui/log";
import { Network, assertNetwork, checkBinary } from "../../utils";
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
        .option("debug", DEBUG_OPTIONS)
        .option("private-key", PRIVATE_KEY_OPTIONS)
        .option("rpc", RPC_OPTIONS);
    },
    async (argv) => {
      checkBinary("sui", "sui");

      const packageDir = argv["package-dir"];
      const network = argv.network.toUpperCase();
      assertNetwork(network);
      const debug = argv.debug ?? false;
      const privateKey = argv["private-key"];
      const rpc = argv.rpc;

      const res = await deploy(network, packageDir, rpc, privateKey);

      // Dump deployment info to console
      logTransactionDigest(res);
      logPublishedPackageId(res);
      if (debug) {
        logTransactionSender(res);
        logCreatedObjects(res);
      }
    }
  );

export const deploy = async (
  network: Network,
  packageDir: string,
  rpc?: string,
  privateKey?: string
): Promise<SuiTransactionBlockResponse> => {
  rpc = rpc ?? NETWORKS[network].sui.rpc;
  const provider = getProvider(network, rpc);
  const signer = getSigner(provider, network, privateKey);

  // Allow absolute paths, otherwise assume relative to directory `worm` command is run from
  const dir = packageDir.startsWith("/")
    ? packageDir
    : `${process.cwd()}/${packageDir}`;
  const packagePath = dir.endsWith("/") ? dir.slice(0, -1) : dir;

  if (!fs.existsSync(packagePath)) {
    throw new Error(
      `Package directory ${packagePath} does not exist. Make sure to deploy from the correct directory or provide an absolute path.`
    );
  }

  return publishPackage(signer, network, packagePath);
};
