import { JsonRpcProvider } from "@mysten/sui.js";
import fs from "fs";
import { Network } from "../../utils";
import { MoveToml } from "./MoveToml";
import {
  buildPackage,
  cleanupTempToml,
  getAllLocalPackageDependencyPaths,
  getDefaultTomlPath,
  getPackageNameFromPath,
  setupMainToml,
} from "./publish";
import { SuiBuildOutput } from "./types";
import { getPackageId } from "./utils";

export const buildCoin = async (
  provider: JsonRpcProvider,
  network: Network,
  packagePath: string,
  coreBridgeStateObjectId: string,
  tokenBridgeStateObjectId: string,
  version: string,
  decimals: number
): Promise<SuiBuildOutput> => {
  const coreBridgePackageId = await getPackageId(
    provider,
    coreBridgeStateObjectId
  );
  const tokenBridgePackageId = await getPackageId(
    provider,
    tokenBridgeStateObjectId
  );
  try {
    setupCoin(
      network,
      packagePath,
      coreBridgePackageId,
      tokenBridgePackageId,
      version,
      decimals
    );
    return buildPackage(`${packagePath}/wrapped_coin`);
  } finally {
    cleanupCoin(`${packagePath}/wrapped_coin`);
  }
};

const setupCoin = (
  network: Network,
  packagePath: string,
  coreBridgePackageId: string,
  tokenBridgePackageId: string,
  version: string,
  decimals: number
): void => {
  // Check to see if the given version string is valid. We don't include the
  // end boundary in the regex to accomodate versions such as V__0_1_0_patch,
  // in the off chance we need such a naming scheme.
  if (!/^V__[0-9]+_[0-9]+_[0-9]+/.test(version)) {
    throw new Error(`Invalid version ${version}`);
  }

  // Assemble package directory
  fs.rmSync(`${packagePath}/wrapped_coin`, { recursive: true, force: true });
  fs.mkdirSync(`${packagePath}/wrapped_coin/sources`, { recursive: true });

  // Replace template variables
  const coinTemplate = fs
    .readFileSync(
      `${packagePath}/templates/wrapped_coin/sources/coin.move`,
      "utf8"
    )
    .toString();
  const coin = coinTemplate
    .replace(/{{DECIMALS}}/, decimals.toString())
    .replace(/{{VERSION}}/g, version);
  fs.writeFileSync(
    `${packagePath}/wrapped_coin/sources/coin.move`,
    coin,
    "utf8"
  );

  // Substitute dependency package IDs
  const toml = new MoveToml(`${packagePath}/templates/wrapped_coin/Move.toml`)
    .updateRow("addresses", "wormhole", coreBridgePackageId)
    .updateRow("addresses", "token_bridge", tokenBridgePackageId)
    .serialize();
  const tomlPath = `${packagePath}/wrapped_coin/Move.toml`;
  fs.writeFileSync(tomlPath, toml, "utf8");

  // Setup dependencies
  const paths = getAllLocalPackageDependencyPaths(tomlPath);
  for (const dependencyPath of paths) {
    // todo(aki): the 4th param is a hack that makes this work, but doesn't
    // necessarily make sense. We should probably revisit this later.
    setupMainToml(dependencyPath, network, false, network !== "DEVNET");
    if (network === "DEVNET") {
      const dependencyToml = new MoveToml(getDefaultTomlPath(dependencyPath));
      switch (getPackageNameFromPath(dependencyPath)) {
        case "wormhole":
          dependencyToml
            .addOrUpdateRow("package", "published-at", coreBridgePackageId)
            .updateRow("addresses", "wormhole", coreBridgePackageId);
          break;
        case "token_bridge":
          dependencyToml
            .addOrUpdateRow("package", "published-at", tokenBridgePackageId)
            .updateRow("addresses", "token_bridge", tokenBridgePackageId);
          break;
        default:
          throw new Error(`Unknown dependency ${dependencyPath}`);
      }
      fs.writeFileSync(
        getDefaultTomlPath(dependencyPath),
        dependencyToml.serialize(),
        "utf8"
      );
    }
  }
};

const cleanupCoin = (packagePath: string) => {
  const paths = getAllLocalPackageDependencyPaths(
    getDefaultTomlPath(packagePath)
  );
  for (const dependencyPath of paths) {
    cleanupTempToml(dependencyPath, false);
  }
};
