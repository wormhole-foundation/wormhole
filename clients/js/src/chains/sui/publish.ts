import { SuiTransactionBlockResponse } from "@mysten/sui/client";
import { Transaction } from "@mysten/sui/transactions";
import { fromB64 } from "@mysten/bcs";
import { execSync } from "child_process";
import fs from "fs";
import { SuiBuildOutput } from "./types";
import {
  executeTransactionBlock,
  isSuiPublishEvent,
  normalizeSuiAddress,
  SuiSigner,
} from "./utils";
import { Network } from "@wormhole-foundation/sdk";

/**
 * Map SDK Network to Sui CLI environment name.
 */
const getEnvironmentFlag = (network: Network): string | undefined => {
  switch (network) {
    case "Mainnet":
      return "mainnet";
    case "Testnet":
      return "testnet";
    case "Devnet":
      return undefined;
    default:
      return undefined;
  }
};

export const buildPackage = (
  packagePath: string,
  network: Network = "Devnet"
): SuiBuildOutput => {
  if (!fs.existsSync(packagePath)) {
    throw new Error(`Package not found at ${packagePath}`);
  }

  const env = getEnvironmentFlag(network);
  const envFlag = env ? `-e ${env}` : "";
  const cmd = `sui move build --dump-bytecode-as-base64 ${envFlag} --path ${packagePath} 2>&1`;

  try {
    const output = execSync(cmd, { encoding: "utf-8" });
    const jsonStart = output.indexOf("{");
    if (jsonStart === -1) {
      throw new Error(`No JSON output from build command: ${output}`);
    }
    return JSON.parse(output.slice(jsonStart));
  } catch (e: any) {
    throw new Error(`Failed to build package: ${e.message}\nCommand: ${cmd}`);
  }
};

/**
 * Publish a package using test-publish for Devnet (ephemeral) or SDK publish for persistent networks.
 */
export const publishPackage = async (
  signer: SuiSigner,
  network: Network,
  packagePath: string
) => {
  if (network === "Devnet") {
    // test-publish uses the locally configured CLI signer, not the passed signer
    return publishPackageTestPublish(packagePath);
  } else {
    return publishPackageSDK(signer, network, packagePath);
  }
};

/**
 * Use `sui client test-publish` for ephemeral/local deployments.
 * This handles dependencies automatically and doesn't require Published.toml manipulation.
 * Note: Uses the locally configured Sui CLI signer, not a programmatic signer.
 */
const publishPackageTestPublish = async (packagePath: string) => {
  // Use test-publish with --publish-unpublished-deps to handle dependencies
  // --build-env testnet tells it to use testnet dependency resolution
  const cmd = `sui client test-publish ${packagePath} --publish-unpublished-deps --build-env testnet --json 2>&1`;

  console.log(`Running: ${cmd}`);

  try {
    const output = execSync(cmd, { encoding: "utf-8" });
    console.log(`test-publish output:\n${output}`);

    // Parse JSON output
    const jsonStart = output.indexOf("{");
    if (jsonStart === -1) {
      throw new Error(`No JSON output from test-publish: ${output}`);
    }

    const result = JSON.parse(output.slice(jsonStart));

    // Extract published package ID from the result
    const publishedChanges = result.objectChanges?.filter(
      (change: any) => change.type === "published"
    );

    if (!publishedChanges || publishedChanges.length === 0) {
      throw new Error(
        `No published package found in test-publish output: ${JSON.stringify(
          result.objectChanges,
          null,
          2
        )}`
      );
    }

    // Find the main package (not dependencies) - it's typically the last one published
    const mainPackage = publishedChanges[publishedChanges.length - 1];
    console.log(`Published package ID: ${mainPackage.packageId}`);

    return result;
  } catch (e: any) {
    // Print full error details
    console.error(`test-publish error:`);
    if (e.stdout) console.error(`stdout: ${e.stdout}`);
    if (e.stderr) console.error(`stderr: ${e.stderr}`);
    if (e.status) console.error(`exit code: ${e.status}`);
    console.error(`message: ${e.message}`);
    throw new Error(`test-publish failed: ${e.message}`);
  }
};

/**
 * Use SDK publish for persistent networks (Mainnet, Testnet).
 */
const publishPackageSDK = async (
  signer: SuiSigner,
  network: Network,
  packagePath: string
) => {
  const build = buildPackage(packagePath, network);

  console.log(
    `Build output: ${build.modules.length} modules, ${build.dependencies.length} dependencies`
  );

  const tx = new Transaction();
  const [upgradeCap] = tx.publish({
    modules: build.modules.map((m) => Array.from(fromB64(m))),
    dependencies: build.dependencies.map((d) => normalizeSuiAddress(d)),
  });

  tx.transferObjects(
    [upgradeCap],
    signer.keypair.getPublicKey().toSuiAddress()
  );

  const res = await executeTransactionBlock(signer, tx);

  console.log(`Transaction status: ${JSON.stringify(res.effects?.status)}`);

  const publishEvents = res.objectChanges?.filter(isSuiPublishEvent) ?? [];
  if (publishEvents.length !== 1) {
    throw new Error(
      "No publish event found in transaction:" +
        JSON.stringify(res.objectChanges, null, 2)
    );
  }

  console.log(`Published package ID: ${publishEvents[0].packageId}`);

  return res;
};
