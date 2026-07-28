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

/**
 * Resolve `packagePath` to a real directory on this system.
 *
 * `realpathSync` resolves symlinks and throws if the path does not exist; the
 * `isDirectory` check then rejects regular files, which the previous
 * `existsSync` test accepted. Passing the *resolved* path on to the CLI also
 * narrows the command-injection surface, since a payload such as
 * `foo; curl evil.com` does not resolve to a real path. Note that this is not
 * a complete injection defense by itself — a directory can legitimately contain
 * shell metacharacters in its name — so the argument is additionally passed via
 * `execFileSync`'s argv rather than through a shell.
 */
const resolvePackageDir = (packagePath: string): string => {
  let resolved: string;
  try {
    resolved = fs.realpathSync(packagePath);
  } catch {
    throw new Error(`Package not found at ${packagePath}`);
  }

  if (!fs.statSync(resolved).isDirectory()) {
    throw new Error(`Package path is not a directory: ${packagePath}`);
  }

  return resolved;
};

export const buildPackage = (
  packagePath: string,
  network: Network = "Devnet"
): SuiBuildOutput => {
  const packageDir = resolvePackageDir(packagePath);

  const env = getEnvironmentFlag(network);
  const envFlag = env ? `-e ${env}` : "";
  const cmd = `sui move build --dump-bytecode-as-base64 -q ${envFlag} --path ${packageDir}`;

  let stdout: string;
  try {
    stdout = execSync(cmd, {
      encoding: "utf-8",
      stdio: ["ignore", "pipe", "pipe"],
    });
  } catch (e: any) {
    const detail = [e.stderr, e.stdout, e.message].filter(Boolean).join("\n");
    throw new Error(`Failed to build package: ${detail}\nCommand: ${cmd}`);
  }

  try {
    return JSON.parse(stdout);
  } catch {
    throw new Error(`Unexpected non-JSON output from build command: ${stdout}`);
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
 * Extract the transaction digest from `sui client test-publish -q --json`
 * output. `-q` keeps the human-readable build lines on stderr, so stdout is a
 * single JSON object and needs no text slicing. Only the digest is read here;
 * the transaction's effects are fetched separately over gRPC.
 */
export const parseTestPublishDigest = (output: string): string => {
  let parsed: any;
  try {
    parsed = JSON.parse(output);
  } catch {
    throw new Error(`Unexpected non-JSON output from test-publish: ${output}`);
  }

  const digest = parsed?.digest;
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
  const packageDir = resolvePackageDir(packagePath);

  // `--build-env testnet` selects testnet dependency pins (localnet is not a
  // pinned environment); `--publish-unpublished-deps` publishes dependencies
  // that are not yet on-chain for this network.
  // `-q` keeps the build progress lines on stderr so stdout carries only the
  // `--json` payload; the streams are left split (no `2>&1`).
  const cmd = `sui client test-publish ${packageDir} --publish-unpublished-deps --build-env testnet -q --json`;

  let output: string;
  try {
    output = execSync(cmd, {
      encoding: "utf-8",
      stdio: ["ignore", "pipe", "pipe"],
    });
  } catch (e: any) {
    const detail = [e.stderr, e.stdout, e.message].filter(Boolean).join("\n");
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
