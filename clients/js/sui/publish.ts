import {
  fromB64,
  getPublishedObjectChanges,
  normalizeSuiObjectId,
  RawSigner,
  TransactionBlock,
} from "@mysten/sui.js";
import { execSync } from "child_process";
import fs from "fs";
import { resolve } from "path";
import { Network } from "../utils";
import { MoveToml } from "./MoveToml";

export const publishPackage = async (
  signer: RawSigner,
  network: Network,
  packagePath: string
) => {
  console.log("Network", network);
  console.log("Package path", packagePath);

  try {
    setupMainToml(packagePath, network);

    // Build contracts
    const buildOutput: {
      modules: string[];
      dependencies: string[];
    } = JSON.parse(
      execSync(
        `sui move build --dump-bytecode-as-base64 --path ${packagePath} 2> /dev/null`,
        {
          encoding: "utf-8",
        }
      )
    );

    // Publish contracts
    const transactionBlock = new TransactionBlock();
    const [upgradeCap] = transactionBlock.publish({
      modules: buildOutput.modules.map((m: string) => Array.from(fromB64(m))),
      dependencies: buildOutput.dependencies.map((d: string) =>
        normalizeSuiObjectId(d)
      ),
    });

    // Transfer upgrade capability to deployer
    transactionBlock.transferObjects(
      [upgradeCap],
      transactionBlock.pure(await signer.getAddress())
    );

    // Execute transactions
    const res = await signer.signAndExecuteTransactionBlock({
      transactionBlock,
      options: {
        showInput: true,
        showObjectChanges: true,
      },
    });

    // Update network-specific Move.toml with package ID
    const publishEvents = getPublishedObjectChanges(res);
    if (publishEvents.length !== 1) {
      throw new Error(
        "No publish event found in transaction:" +
          JSON.stringify(res.objectChanges, null, 2)
      );
    }

    updateNetworkToml(packagePath, network, publishEvents[0].packageId);

    // Return publish transaction info
    return res;
  } catch (e) {
    throw e;
  } finally {
    cleanupTempToml(packagePath);
  }
};

const cleanupTempToml = (packagePath: string): void => {
  const defaultTomlPath = getDefaultTomlPath(packagePath);
  const tempTomlPath = getTempTomlPath(packagePath);
  if (fs.existsSync(tempTomlPath)) {
    // Clean up Move.toml for dependencies
    const dependencyPaths = getAllLocalPackageDependencyPaths(defaultTomlPath);
    for (const path of dependencyPaths) {
      cleanupTempToml(path);
    }

    fs.renameSync(tempTomlPath, defaultTomlPath);
  }
};

/**
 * Get Move.toml dependencies by looking for all lines of form 'local = ".*"'.
 * This works because network-specific Move.toml files should not contain
 * dev addresses, so the only lines that match this regex are the dependencies
 * that need to be replaced.
 * @param packagePath
 * @returns
 */
const getAllLocalPackageDependencyPaths = (tomlPath: string): string[] => {
  const tomlStr = fs.readFileSync(tomlPath, "utf8").toString();
  const toml = new MoveToml(tomlStr);

  // Sanity check that Move.toml does not contain dev info since this breaks
  // building and publishing packages
  if (
    toml.getSectionNames().some((name) => name.includes("dev-dependencies")) ||
    toml.getSectionNames().some((name) => name.includes("dev-addresses"))
  ) {
    throw new Error(
      "Network-specific Move.toml should not contain dev-dependencies or dev-addresses."
    );
  }

  const packagePath = getPackagePathFromTomlPath(tomlPath);
  return [...tomlStr.matchAll(/local = "(.*)"/g)].map((match) =>
    resolve(packagePath, match[1])
  );
};

const getDefaultTomlPath = (packagePath: string): string =>
  `${packagePath}/Move.toml`;

const getPackagePathFromTomlPath = (tomlPath: string): string =>
  tomlPath.split("/").slice(0, -1).join("/");

const getTempTomlPath = (packagePath: string): string =>
  `${packagePath}/Move.temp.toml`;

const getTomlPathByNetwork = (packagePath: string, network: Network): string =>
  `${packagePath}/Move.${network.toLowerCase()}.toml`;

const getPackageNameFromPath = (packagePath: string): string =>
  packagePath.split("/").pop() || "";

const resetNetworkToml = (
  packagePath: string,
  network: Network,
  recursive: boolean = false
): void => {
  const networkTomlPath = getTomlPathByNetwork(packagePath, network);
  const tomlStr = fs.readFileSync(networkTomlPath, "utf8").toString();
  const toml = new MoveToml(tomlStr);
  if (toml.isPublished()) {
    if (recursive) {
      const dependencyPaths =
        getAllLocalPackageDependencyPaths(networkTomlPath);
      for (const path of dependencyPaths) {
        resetNetworkToml(path, network);
      }
    }

    const updatedTomlStr = toml
      .removeRow("package", "published-at")
      .updateRow("addresses", getPackageNameFromPath(packagePath), "_")
      .serialize();
    fs.writeFileSync(networkTomlPath, updatedTomlStr, "utf8");
  }
};

const setupMainToml = (
  packagePath: string,
  network: Network,
  isDependency: boolean = false
): void => {
  const defaultTomlPath = getDefaultTomlPath(packagePath);
  const tempTomlPath = getTempTomlPath(packagePath);
  const srcTomlPath = getTomlPathByNetwork(packagePath, network);

  if (fs.existsSync(tempTomlPath)) {
    // It's possible that this dependency has been set up by another package
    if (isDependency) {
      return;
    }

    throw new Error("Move.temp.toml exists, is there a publish in progress?");
  }

  // Make deploying on devnet more convenient by resetting Move.toml so we
  // don't have to manually reset them repeatedly during local development.
  // This is not recursive because we assume that packages are deployed bottom
  // up.
  if (!isDependency && network === "DEVNET") {
    resetNetworkToml(packagePath, network);
  }

  // Save default Move.toml
  if (!fs.existsSync(defaultTomlPath)) {
    throw new Error(
      `Invalid package layout. Move.toml not found at ${defaultTomlPath}`
    );
  }

  fs.renameSync(defaultTomlPath, tempTomlPath);

  // Set Move.toml from appropriate network
  if (!fs.existsSync(srcTomlPath)) {
    throw new Error(`Move.toml for ${network} not found at ${srcTomlPath}`);
  }

  fs.copyFileSync(srcTomlPath, defaultTomlPath);

  // Replace undefined addresses in base Move.toml and ensure dependencies are
  // published
  const tomlStr = fs.readFileSync(defaultTomlPath, "utf8").toString();
  const toml = new MoveToml(tomlStr);
  const packageName = getPackageNameFromPath(packagePath);
  if (!isDependency) {
    if (toml.isPublished()) {
      throw new Error(`Package ${packageName} is already published.`);
    } else {
      toml.updateRow("addresses", packageName, "0x0");
    }

    fs.writeFileSync(defaultTomlPath, toml.serialize());
  } else if (isDependency && !toml.isPublished()) {
    throw new Error(
      `Dependency ${packageName} is not published. Please publish it first.`
    );
  }

  // Set up Move.toml for dependencies
  const dependencyPaths = getAllLocalPackageDependencyPaths(defaultTomlPath);
  for (const path of dependencyPaths) {
    setupMainToml(path, network, true);
  }
};

const updateNetworkToml = (
  packagePath: string,
  network: Network,
  packageId: string
): void => {
  const tomlPath = getTomlPathByNetwork(packagePath, network);
  const tomlStr = fs.readFileSync(tomlPath, "utf8");
  const updatedTomlStr = new MoveToml(tomlStr)
    .addRow("package", "published-at", packageId)
    .updateRow("addresses", getPackageNameFromPath(packagePath), packageId)
    .serialize();
  fs.writeFileSync(tomlPath, updatedTomlStr, "utf8");
};
