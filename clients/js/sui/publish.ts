import {
  fromB64,
  normalizeSuiObjectId,
  RawSigner,
  TransactionBlock,
} from "@mysten/sui.js";
import { execSync } from "child_process";
import fs from "fs";
import { resolve } from "path";
import { isSuiCreateEvent, isSuiPublishEvent } from ".";
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
    setupToml(packagePath, network);

    // Build contracts
    const buildOutput: {
      modules: string[];
      dependencies: string[];
    } = JSON.parse(
      execSync(
        `sui move build --dump-bytecode-as-base64 --path ${packagePath}`,
        {
          encoding: "utf-8",
        }
      )
    );

    // Publish contracts
    const transactionBlock = new TransactionBlock();
    const [upgradeCap] = transactionBlock.publish(
      buildOutput.modules.map((m: string) => Array.from(fromB64(m))),
      buildOutput.dependencies.map((d: string) => normalizeSuiObjectId(d))
    );

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
    const tomlPath = getTomlPathByNetwork(packagePath, network);
    const tomlStr = fs.readFileSync(tomlPath, "utf8");
    const publishEvent = res.objectChanges.find(isSuiPublishEvent);
    if (!publishEvent) {
      throw new Error(
        "No publish event found in transaction:" +
          JSON.stringify(res.objectChanges, null, 2)
      );
    }

    const updatedTomlStr = new MoveToml(tomlStr)
      .addRow("package", "published-at", publishEvent.packageId)
      .updateRow(
        "addresses",
        getPackageNameFromPath(packagePath),
        publishEvent.packageId
      )
      .serialize();
    fs.writeFileSync(tomlPath, updatedTomlStr, "utf8");

    // Dump deployment info to console
    console.log("Transaction digest", res.digest);
    console.log("Deployer", res.transaction.data.sender);
    console.log("Published to", publishEvent.packageId);
    console.log(
      "Created objects",
      res.objectChanges.filter(isSuiCreateEvent).map((e) => {
        return {
          type: e.objectType,
          objectId: e.objectId,
          owner: e.owner["AddressOwner"] || e.owner["ObjectOwner"] || e.owner,
        };
      })
    );

    // Return publish transaction info
    return res;
  } catch (e) {
    throw e;
  } finally {
    cleanupToml(packagePath);
  }
};

const cleanupToml = (packagePath: string): void => {
  const defaultTomlPath = getDefaultTomlPath(packagePath);
  const tempTomlPath = getTempTomlPath(packagePath);
  if (fs.existsSync(tempTomlPath)) {
    // Clean up Move.toml for dependencies
    const dependencyPaths = getAllLocalPackageDependencyPaths(packagePath);
    for (const path of dependencyPaths) {
      cleanupToml(path);
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
const getAllLocalPackageDependencyPaths = (packagePath: string): string[] => {
  const tomlPath = getDefaultTomlPath(packagePath);
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

  return [...tomlStr.matchAll(/local = "(.*)"/g)].map((match) =>
    resolve(packagePath, match[1])
  );
};

const getDefaultTomlPath = (packagePath: string): string =>
  `${packagePath}/Move.toml`;

const getTempTomlPath = (packagePath: string): string =>
  `${packagePath}/Move.temp.toml`;

const getTomlPathByNetwork = (packagePath: string, network: Network): string =>
  `${packagePath}/Move.${network.toLowerCase()}.toml`;

const getPackageNameFromPath = (packagePath: string): string =>
  packagePath.split("/").pop() || "";

const setupToml = (
  packagePath: string,
  network: Network,
  isDependency: boolean = false
): void => {
  const defaultTomlPath = getDefaultTomlPath(packagePath);
  const tempTomlPath = getTempTomlPath(packagePath);

  if (fs.existsSync(tempTomlPath)) {
    // It's possible that this dependency has been set up by another package
    if (isDependency) {
      return;
    }

    throw new Error("Move.temp.toml exists, is there a publish in progress?");
  }

  // Save default Move.toml
  if (!fs.existsSync(defaultTomlPath)) {
    throw new Error(
      `Invalid package layout. Move.toml not found at ${defaultTomlPath}`
    );
  }

  fs.renameSync(defaultTomlPath, tempTomlPath);

  // Set Move.toml from appropriate network
  const srcTomlPath = getTomlPathByNetwork(packagePath, network);
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
  const dependencyPaths = getAllLocalPackageDependencyPaths(packagePath);
  for (const path of dependencyPaths) {
    setupToml(path, network, true);
  }
};
