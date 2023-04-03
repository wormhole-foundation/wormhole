import {
  Connection,
  Ed25519Keypair,
  fromB64,
  JsonRpcProvider,
  normalizeSuiObjectId,
  RawSigner,
  TransactionBlock,
} from "@mysten/sui.js";
import { execSync } from "child_process";
import fs from "fs";
import { resolve } from "path";
import { NETWORKS } from "./networks";
import { Network } from "./utils";

export async function executeTransactionBlock(
  provider: JsonRpcProvider,
  network: Network,
  transactionBlock: TransactionBlock
) {
  const signer = getSigner(provider, network);
  const testRes = await signer.dryRunTransactionBlock({ transactionBlock });
  if (testRes.effects.status.status !== "success") {
    throw new Error(
      `Failed to execute transaction: ${testRes.effects.status.error}`
    );
  }

  const res = await signer.signAndExecuteTransactionBlock({ transactionBlock });
  console.log("Digest", res.digest, res.effects.transactionDigest);
  console.log("Sender", res.transaction.data.sender);

  // console.log(
  //   "Transaction digest: ",
  //   moveCallTxn["certificate"]["transactionDigest"]
  // );
  // console.log(
  //   "Sender:             ",
  //   moveCallTxn["certificate"]["data"]["sender"]
  // );

  // Let caller handle parsing and logging effects
  // return moveCallTxn["effects"]["effects"];
  return res;
}

export const getOwnedObjectId = async (
  provider: JsonRpcProvider,
  owner: string,
  packageId: string,
  moduleName: string,
  structName: string
): Promise<string | null> => {
  const res = await provider.getOwnedObjects({
    owner,
    filter: { StructType: `${packageId}::${moduleName}::${structName}` },
  });
  return res.data.length > 0 ? res.data[0].data.objectId : null;
};

export const getProvider = (
  network?: Network,
  rpc?: string
): JsonRpcProvider => {
  if (!network && !rpc) {
    throw new Error("Must provide network or RPC to initialize provider");
  }

  rpc = rpc || NETWORKS[network]["sui"].rpc;
  if (!rpc) {
    throw new Error(`No default RPC found for Sui ${network}`);
  }

  return new JsonRpcProvider(new Connection({ fullnode: rpc }));
};

export const getSigner = (
  provider: JsonRpcProvider,
  network: Network
): RawSigner => {
  const privateKey: string | undefined = NETWORKS[network]["sui"].key;
  if (!privateKey) {
    throw new Error(`No private key found for Sui ${network}`);
  }

  const bytes = fromB64(privateKey);
  const keypair = Ed25519Keypair.fromSecretKey(bytes.slice(1));
  return new RawSigner(keypair, provider);
};

export const isValidSuiObjectId = (objectId: string): boolean => {
  return /^(0x)?[0-9a-f]{64}$/i.test(objectId);
};

export type SuiPublishEvent = {
  packageId: string;
  type: "published";
  version: number;
  digest: string;
  modules: string[];
};

export const isSuiPublishEvent = (event: any): event is SuiPublishEvent => {
  return event.type === "published";
};

export type SuiCreateEvent = {
  sender: string;
  type: "created";
  objectType: string;
  objectId: string;
  version: number;
  digest: string;
  owner:
    | {
        AddressOwner: string;
      }
    | {
        ObjectOwner: string;
      }
    | {
        Shared: {
          initial_shared_version: number;
        };
      }
    | "Immutable";
};

export const isSuiCreateEvent = (event: any): event is SuiCreateEvent => {
  return event.type === "created";
};

export const publishPackage = async (
  provider: JsonRpcProvider,
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
    const signer = getSigner(provider, network);
    transactionBlock.transferObjects(
      [upgradeCap],
      transactionBlock.pure(await signer.getAddress())
    );

    // Execute transactions
    transactionBlock.setGasBudget(10000000);
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

  // Sanity check that Move.toml does not contain dev info since this currently
  // breaks building and publishing packages
  if (
    /\[dev\-dependencies\]/.test(tomlStr) ||
    /\[dev\-addresses\]/.test(tomlStr)
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

type ParsedMoveToml = {
  name: string;
  rows: { key: string; value: string }[];
}[];

class MoveToml {
  toml: ParsedMoveToml;

  constructor(tomlStr: string) {
    this.toml = MoveToml.parse(tomlStr);
  }

  addRow(sectionName: string, key: string, value: string) {
    if (!MoveToml.isValidValue(value)) {
      if (/^\S+$/.test(value)) {
        value = `"${value}"`;
      } else {
        throw new Error(`Invalid value "${value}"`);
      }
    }

    const section = this.toml.find((s) => s.name === sectionName);
    if (section === undefined) {
      throw new Error(`Section "${sectionName}" not found`);
    }

    section.rows.push({ key, value });
    return this;
  }

  isPublished(): boolean {
    const section = this.toml.find((s) => s.name === "package");
    if (section === undefined) {
      throw new Error('Section "package" not found');
    }

    const row = section.rows.find((r) => r.key === "published-at");
    return row !== undefined;
  }

  serialize(): string {
    let tomlStr = "";
    for (let i = 0; i < this.toml.length; i++) {
      const section = this.toml[i];
      tomlStr += `[${section.name}]\n`;
      for (const row of section.rows) {
        tomlStr += `${row.key} = ${row.value}\n`;
      }

      if (i !== this.toml.length - 1) {
        tomlStr += "\n";
      }
    }

    return tomlStr;
  }

  updateRow(sectionName: string, key: string, value: string) {
    if (!MoveToml.isValidValue(value)) {
      if (/^\S+$/.test(value)) {
        value = `"${value}"`;
      } else {
        throw new Error(`Invalid value "${value}"`);
      }
    }

    const section = this.toml.find((s) => s.name === sectionName);
    if (section === undefined) {
      throw new Error(`Section "${sectionName}" not found`);
    }

    const row = section.rows.find((r) => r.key === key);
    if (row === undefined) {
      throw new Error(`Row "${key}" not found in section "${sectionName}"`);
    }

    row.value = value;
    return this;
  }

  static isValidValue(value: string): boolean {
    value = value.trim();
    return (
      (value.startsWith('"') && value.endsWith('"')) ||
      (value.startsWith("{") && value.endsWith("}")) ||
      (value.startsWith("'") && value.endsWith("'"))
    );
  }

  static parse(tomlStr: string): ParsedMoveToml {
    const toml: ParsedMoveToml = [];
    const lines = tomlStr.split("\n");
    for (const line of lines) {
      // Parse new section
      const sectionMatch = line.trim().match(/^\[(\S+)\]$/);
      if (sectionMatch && sectionMatch.length === 2) {
        toml.push({ name: sectionMatch[1], rows: [] });
        continue;
      }

      // Otherwise, parse row in section. We must handle two cases:
      //  1. value is string, e.g. name = "MyPackage"
      //  2. value is object, e.g. Sui = { local = "../sui-framework" }
      const rowMatch = line.trim().match(/^([a-zA-Z_\-]+) = (.+)$/);
      if (rowMatch && rowMatch.length === 3) {
        toml[toml.length - 1].rows.push({
          key: rowMatch[1],
          value: rowMatch[2],
        });
      }
    }

    return toml;
  }
}
