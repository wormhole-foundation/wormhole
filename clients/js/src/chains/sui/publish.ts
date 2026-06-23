import { Transaction } from "@mysten/sui/transactions";
import { fromBase64 } from "@mysten/sui/utils";
import { execSync } from "child_process";
import fs from "fs";
import { SuiBuildOutput } from "./types";
import {
  executeTransactionBlock,
  fetchTransactionResult,
  getPublishedPackageId,
  normalizeSuiAddress,
  SuiSigner,
  SuiTransactionResult,
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
): Promise<SuiTransactionResult> => {
  if (network === "Devnet") {
    return publishPackageTestPublish(signer, packagePath);
  }
  return publishPackageSDK(signer, network, packagePath);
};

/**
 * Extract the transaction digest from `sui client test-publish --json` output.
 * The CLI prepends human-readable build lines before the JSON object, so the
 * payload is sliced from the first brace. Only the digest is read here; the
 * transaction's effects are fetched separately over gRPC.
 */
export const parseTestPublishDigest = (output: string): string => {
  const jsonStart = output.indexOf("{");
  if (jsonStart === -1) {
    throw new Error(`No JSON output from test-publish: ${output}`);
  }
  const digest = JSON.parse(output.slice(jsonStart))?.digest;
  if (typeof digest !== "string") {
    throw new Error(`No transaction digest in test-publish output: ${output}`);
  }
  return digest;
};

/**
 * Publish via `sui client test-publish` for Devnet/localnet.
 *
 * The Sui package system resolves local, not-yet-published dependencies (e.g.
 * the core bridge imported by the token bridge) per chain id. `test-publish`
 * performs that resolution together with `--publish-unpublished-deps` without
 * requiring Published.toml management — something a plain SDK `tx.publish`
 * cannot reproduce on an ephemeral local network. It signs with the locally
 * configured CLI keystore and commits on-chain, so the resulting transaction is
 * read back over gRPC rather than parsed from the CLI's deprecated JSON-RPC
 * `objectChanges` payload.
 */
const publishPackageTestPublish = async (
  signer: SuiSigner,
  packagePath: string
): Promise<SuiTransactionResult> => {
  // `--build-env testnet` selects testnet dependency pins (localnet is not a
  // pinned environment); `--publish-unpublished-deps` publishes dependencies
  // that are not yet on-chain for this network.
  const cmd = `sui client test-publish ${packagePath} --publish-unpublished-deps --build-env testnet --json 2>&1`;

  let output: string;
  try {
    output = execSync(cmd, { encoding: "utf-8" });
  } catch (e: any) {
    const detail = [e.stdout, e.stderr, e.message].filter(Boolean).join("\n");
    throw new Error(`test-publish failed:\n${detail}`);
  }

  const digest = parseTestPublishDigest(output);
  return fetchTransactionResult(signer.client, digest);
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
    modules: build.modules.map((m) => Array.from(fromBase64(m))),
    dependencies: build.dependencies.map((d) => normalizeSuiAddress(d)),
  });

  tx.transferObjects(
    [upgradeCap],
    signer.keypair.getPublicKey().toSuiAddress()
  );

  const res = await executeTransactionBlock(signer, tx);

  console.log(`Transaction status: ${res.success ? "success" : res.error}`);

  // getPublishedPackageId throws if there isn't exactly one published package.
  console.log(`Published package ID: ${getPublishedPackageId(res)}`);

  return res;
};
