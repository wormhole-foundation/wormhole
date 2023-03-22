import {
  Connection,
  Ed25519Keypair,
  JsonRpcProvider,
  RawSigner,
} from "@mysten/sui.js";
import { execSync } from "child_process";
import fs from "fs";
import { resolve } from "path";
import { NETWORKS } from "./networks";
import { Network } from "./utils";

export async function executeEntry(
  provider: JsonRpcProvider,
  network: Network,
  packageObjectId: string,
  moduleName: string,
  functionName: string,
  typeArgs: string[],
  args: any[]
) {
  const signer = getSigner(provider, network);
  const moveCallTxn = await signer.executeMoveCall({
    packageObjectId,
    module: moduleName,
    function: functionName,
    typeArguments: typeArgs,
    arguments: args,
    gasBudget: 50000,
  });

  console.log(
    "Transaction digest: ",
    moveCallTxn["certificate"]["transactionDigest"]
  );
  console.log(
    "Sender:             ",
    moveCallTxn["certificate"]["data"]["sender"]
  );

  // Let caller handle parsing and logging effects
  return moveCallTxn["effects"]["effects"];
}

export const getObjectFromOwner = async (
  provider: JsonRpcProvider,
  owner: string,
  packageId: string,
  moduleName: string,
  objectName: string
): Promise<string | null> => {
  const objects = await provider.getObjectsOwnedByAddress(owner);
  const type = `${packageId}::${moduleName}::${objectName}`;
  return (
    objects.find((o) => o.type.toLowerCase() === type.toLowerCase())
      ?.objectId ?? null
  );
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

  const bytes = new Uint8Array(Buffer.from(privateKey, "base64"));
  const keypair = Ed25519Keypair.fromSecretKey(bytes.slice(1));
  return new RawSigner(keypair, provider);
};

export const publishPackage = async (
  provider: JsonRpcProvider,
  network: Network,
  packagePath: string,
  namedAddresses: { [key: string]: string }
) => {
  console.log("Network:         ", network);
  console.log("Package path:    ", packagePath);
  console.log("Named addresses: ", JSON.stringify(namedAddresses, null, 2));

  try {
    setupToml(packagePath, network, namedAddresses);

    // Build contracts
    const compiledModules: string[] = JSON.parse(
      execSync(
        `sui move build --dump-bytecode-as-base64 --path ${packagePath}`,
        {
          encoding: "utf-8",
        }
      )
    );

    // Publish contracts
    const signer = getSigner(provider, network);
    const publishTx = await signer.publish({
      compiledModules,
      gasBudget: 1000000,
    });

    // Dump deployment info to console
    console.log(
      "Transaction digest: ",
      publishTx["certificate"]["transactionDigest"]
    );
    console.log(
      "Deployer:           ",
      publishTx["certificate"]["data"]["sender"]
    );
    console.log(
      "Deployed to:        ",
      publishTx["effects"]["effects"]["created"].find(
        (o) => o.owner === "Immutable"
      )["reference"]["objectId"]
    );
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
    const dependencyPaths = getAllPackageDependencyPaths(packagePath);
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
const getAllPackageDependencyPaths = (packagePath: string): string[] => {
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
  namedAddresses: { [key: string]: string },
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

  // Replace named addresses
  let tomlStr = fs.readFileSync(defaultTomlPath, "utf8").toString();
  if (isDependency) {
    for (const [name, address] of Object.entries(namedAddresses)) {
      tomlStr = tomlStr.replace(
        new RegExp(`${name} = "_"`, "g"),
        `${name} = "${address}"`
      );
    }
  } else {
    const name = getPackageNameFromPath(packagePath);
    tomlStr = tomlStr.replace(
      new RegExp(`${name} = "_"`, "g"),
      `${name} = "0x0"`
    );
  }

  fs.writeFileSync(defaultTomlPath, tomlStr);

  // Set up Move.toml for dependencies
  const dependencyPaths = getAllPackageDependencyPaths(packagePath);
  for (const path of dependencyPaths) {
    setupToml(path, network, namedAddresses, true);
  }
};
