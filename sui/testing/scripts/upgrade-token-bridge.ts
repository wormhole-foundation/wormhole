/**
 * Testnet Sui Token Bridge upgrade via governance VAA — gRPC edition.
 *
 * Usage:
 *   TESTNET_WALLET_PRIVATE_KEY="suiprivkey1..." \
 *   TESTNET_GUARDIAN_PRIVATE_KEY="<hex>" \
 *   npx tsx scripts/upgrade-token-bridge-grpc.ts
 *
 *   npx tsx scripts/upgrade-token-bridge-grpc.ts --build-only
 *     Builds, prints the digest, fetches package ids over gRPC, and constructs
 *     the upgrade PTB without signing or executing anything. No keys needed.
 *
 * TESTNET_WALLET_PRIVATE_KEY accepts the bech32 `suiprivkey1...` export
 * directly, or the legacy base64 (scheme-byte-prefixed) format.
 */
import { SuiGrpcClient } from "@mysten/sui/grpc";
import { Transaction } from "@mysten/sui/transactions";
import { Ed25519Keypair } from "@mysten/sui/keypairs/ed25519";
import { decodeSuiPrivateKey } from "@mysten/sui/cryptography";
import { SUI_CLOCK_OBJECT_ID, normalizeSuiObjectId } from "@mysten/sui/utils";
import { secp256k1 } from "@noble/curves/secp256k1";
import { keccak_256 } from "@noble/hashes/sha3";
import { execSync } from "child_process";
import { resolve } from "path";

const TOKEN_BRIDGE_STATE_ID =
  "0x6fb10cdb7aa299e9a4308752dadecb049ff55a892de92992a1edbd7912b3d6da";
const WORMHOLE_STATE_ID =
  "0x31358d198147da50db32eda2562951d53973a0c0ad5ed738e9b17d88b213d790";

const GOVERNANCE_EMITTER = Buffer.from(
  "0000000000000000000000000000000000000000000000000000000000000004",
  "hex",
);
const GOVERNANCE_CHAIN = 1;
const SUI_CHAIN_ID = 21;
const ACTION_UPGRADE_CONTRACT = 2;

const GRPC_URL =
  process.env.SUI_GRPC_URL ?? "https://fullnode.testnet.sui.io:443";

async function main() {
  const buildOnly = process.argv.includes("--build-only");

  const client = new SuiGrpcClient({ network: "testnet", baseUrl: GRPC_URL });

  // 1. Build for bytecode and digest (addresses resolve from Published.toml).
  const tokenBridgePath = resolve(`${__dirname}/../../token_bridge`);
  const { modules, dependencies, digest } =
    buildForBytecodeAndDigest(tokenBridgePath);
  console.log("build digest :", digest.toString("hex"));
  console.log("dependencies :", dependencies);

  // Current package ids from the shared state objects (upgrade_cap.package).
  const tokenBridgePackage = await getPackageId(client, TOKEN_BRIDGE_STATE_ID);
  const wormholePackage = await getPackageId(client, WORMHOLE_STATE_ID);
  console.log("token bridge :", tokenBridgePackage);
  console.log("wormhole     :", wormholePackage);

  if (buildOnly) {
    // Construct (but do not sign/execute) the upgrade PTB to validate the flow.
    buildUpgradeTx(
      wormholePackage,
      tokenBridgePackage,
      modules,
      dependencies,
      Buffer.alloc(1), // placeholder VAA; PTB construction is offline
    );
    console.log("\n--build-only: upgrade PTB constructed OK, exiting.");
    return;
  }

  const guardianKeyHex = process.env.TESTNET_GUARDIAN_PRIVATE_KEY;
  if (!guardianKeyHex) {
    throw new Error("TESTNET_GUARDIAN_PRIVATE_KEY unset in environment");
  }
  const walletKey = process.env.TESTNET_WALLET_PRIVATE_KEY;
  if (!walletKey) {
    throw new Error("TESTNET_WALLET_PRIVATE_KEY unset in environment");
  }
  const signer = loadKeypair(walletKey);
  console.log("wallet       :", signer.toSuiAddress());

  // 2. Sign the UpgradeContract governance VAA over the build digest.
  const signedVaa = makeUpgradeVaa(digest, guardianKeyHex);
  console.log("\nUpgrade VAA  :", signedVaa.toString("hex"));

  // 3. Execute the upgrade.
  const upgradeTx = buildUpgradeTx(
    wormholePackage,
    tokenBridgePackage,
    modules,
    dependencies,
    signedVaa,
  );
  const upgradeResult = await client.signAndExecuteTransaction({
    transaction: upgradeTx,
    signer,
  });
  if (upgradeResult.FailedTransaction) {
    throw new Error(
      `upgrade failed: ${JSON.stringify(upgradeResult.FailedTransaction)}`,
    );
  }
  console.log("\nupgrade tx   :", upgradeResult.Transaction!.digest);

  // 4. Wait until the state object reports the new package.
  const newPackage = await waitForNewPackage(client, tokenBridgePackage);
  console.log("new package  :", newPackage);

  // 5. Migrate — flips the active version (V__0_2_0 → V__0_3_0 dynamic field).
  const migrateTx = buildMigrateTx(wormholePackage, newPackage, signedVaa);
  const migrateResult = await client.signAndExecuteTransaction({
    transaction: migrateTx,
    signer,
  });
  if (migrateResult.FailedTransaction) {
    throw new Error(
      `migrate failed: ${JSON.stringify(migrateResult.FailedTransaction)}`,
    );
  }
  console.log("migrate tx   :", migrateResult.Transaction!.digest);
  console.log(
    "\nDone. Update sui/token_bridge/Published.toml [published.testnet]:" +
      `\n  published-at = "${newPackage}"\n  version bump, then PR that file only.`,
  );
}

function loadKeypair(key: string): Ed25519Keypair {
  if (key.startsWith("suiprivkey")) {
    // fromSecretKey accepts the bech32 export directly (ed25519 only).
    const { scheme } = decodeSuiPrivateKey(key);
    if (scheme !== "ED25519") {
      throw new Error(`unsupported key scheme ${scheme}`);
    }
    return Ed25519Keypair.fromSecretKey(key);
  }
  // Legacy format: base64 with a leading scheme byte.
  return Ed25519Keypair.fromSecretKey(Buffer.from(key, "base64").subarray(1));
}

function buildForBytecodeAndDigest(packagePath: string) {
  const buildOutput: {
    modules: string[];
    dependencies: string[];
    digest: number[];
  } = JSON.parse(
    execSync(
      `sui move build --dump-bytecode-as-base64 -e testnet -p ${packagePath} 2> /dev/null`,
      { encoding: "utf-8", maxBuffer: 64 * 1024 * 1024 },
    ),
  );
  return {
    modules: buildOutput.modules, // base64 strings, accepted by tx.upgrade
    dependencies: buildOutput.dependencies.map((d) => normalizeSuiObjectId(d)),
    digest: Buffer.from(buildOutput.digest),
  };
}

/// UpgradeContract governance VAA: module "TokenBridge", action 2, chain 21,
/// payload = 32-byte build digest. Signed by the single testnet guardian
/// (guardian set index 0).
function makeUpgradeVaa(digest: Buffer, guardianKeyHex: string): Buffer {
  if (digest.length !== 32) throw new Error("digest must be 32 bytes");

  const payload = Buffer.alloc(32 + 1 + 2 + 32);
  payload.write("TokenBridge", 32 - "TokenBridge".length, "ascii");
  payload.writeUInt8(ACTION_UPGRADE_CONTRACT, 32);
  payload.writeUInt16BE(SUI_CHAIN_ID, 33);
  digest.copy(payload, 35);

  const nowSec = Math.floor(Date.now() / 1000);
  const body = Buffer.alloc(4 + 4 + 2 + 32 + 8 + 1 + payload.length);
  let o = 0;
  o = body.writeUInt32BE(nowSec, o); // timestamp
  o = body.writeUInt32BE(0, o); // nonce
  o = body.writeUInt16BE(GOVERNANCE_CHAIN, o);
  o += GOVERNANCE_EMITTER.copy(body, o);
  o = body.writeBigUInt64BE(BigInt(nowSec), o); // sequence (unique per run)
  o = body.writeUInt8(0, o); // consistency level
  payload.copy(body, o);

  const sigDigest = keccak_256(keccak_256(body));
  const sig = secp256k1.sign(sigDigest, Buffer.from(guardianKeyHex, "hex"));

  const vaa = Buffer.alloc(1 + 4 + 1 + 66 + body.length);
  let p = 0;
  p = vaa.writeUInt8(1, p); // version
  p = vaa.writeUInt32BE(0, p); // guardian set index
  p = vaa.writeUInt8(1, p); // num signatures
  p = vaa.writeUInt8(0, p); // guardian index
  p += Buffer.from(sig.toCompactRawBytes()).copy(vaa, p); // r || s
  p = vaa.writeUInt8(sig.recovery, p); // recovery id
  body.copy(vaa, p);
  return vaa;
}

async function getPackageId(
  client: SuiGrpcClient,
  stateId: string,
): Promise<string> {
  const { object } = await client.core.getObject({
    objectId: stateId,
    include: { json: true },
  });
  const json = object.json as { upgrade_cap?: { package?: string } } | null;
  const pkg = json?.upgrade_cap?.package;
  if (!pkg) {
    throw new Error(`upgrade_cap.package not found on ${stateId}`);
  }
  return normalizeSuiObjectId(pkg);
}

async function waitForNewPackage(
  client: SuiGrpcClient,
  oldPackage: string,
): Promise<string> {
  for (let i = 0; i < 30; i++) {
    await new Promise((r) => setTimeout(r, 2000));
    const pkg = await getPackageId(client, TOKEN_BRIDGE_STATE_ID);
    if (pkg !== oldPackage) return pkg;
  }
  throw new Error(
    "state object still reports the old package after 60s — check the upgrade tx",
  );
}

/// parse_and_verify → authorize_governance → verify_vaa → authorize_upgrade →
/// Upgrade → commit_upgrade, all against the CURRENT package.
function buildUpgradeTx(
  wormholePackage: string,
  tokenBridgePackage: string,
  modules: string[],
  dependencies: string[],
  signedVaa: Buffer,
): Transaction {
  const tx = new Transaction();

  const [verifiedVaa] = tx.moveCall({
    target: `${wormholePackage}::vaa::parse_and_verify`,
    arguments: [
      tx.object(WORMHOLE_STATE_ID),
      tx.pure.vector("u8", Array.from(signedVaa)),
      tx.object(SUI_CLOCK_OBJECT_ID),
    ],
  });
  const [decreeTicket] = tx.moveCall({
    target: `${tokenBridgePackage}::upgrade_contract::authorize_governance`,
    arguments: [tx.object(TOKEN_BRIDGE_STATE_ID)],
  });
  const [decreeReceipt] = tx.moveCall({
    target: `${wormholePackage}::governance_message::verify_vaa`,
    arguments: [tx.object(WORMHOLE_STATE_ID), verifiedVaa, decreeTicket],
    typeArguments: [
      `${tokenBridgePackage}::upgrade_contract::GovernanceWitness`,
    ],
  });
  const [upgradeTicket] = tx.moveCall({
    target: `${tokenBridgePackage}::upgrade_contract::authorize_upgrade`,
    arguments: [tx.object(TOKEN_BRIDGE_STATE_ID), decreeReceipt],
  });
  const [upgradeReceipt] = tx.upgrade({
    modules,
    dependencies,
    package: tokenBridgePackage,
    ticket: upgradeTicket,
  });
  tx.moveCall({
    target: `${tokenBridgePackage}::upgrade_contract::commit_upgrade`,
    arguments: [tx.object(TOKEN_BRIDGE_STATE_ID), upgradeReceipt],
  });

  return tx;
}

/// parse_and_verify → authorize_governance → verify_vaa → migrate, against the
/// NEW package (the state object's upgrade_cap points at it post-upgrade).
function buildMigrateTx(
  wormholePackage: string,
  newTokenBridgePackage: string,
  signedVaa: Buffer,
): Transaction {
  const tx = new Transaction();

  const [verifiedVaa] = tx.moveCall({
    target: `${wormholePackage}::vaa::parse_and_verify`,
    arguments: [
      tx.object(WORMHOLE_STATE_ID),
      tx.pure.vector("u8", Array.from(signedVaa)),
      tx.object(SUI_CLOCK_OBJECT_ID),
    ],
  });
  const [decreeTicket] = tx.moveCall({
    target: `${newTokenBridgePackage}::upgrade_contract::authorize_governance`,
    arguments: [tx.object(TOKEN_BRIDGE_STATE_ID)],
  });
  const [decreeReceipt] = tx.moveCall({
    target: `${wormholePackage}::governance_message::verify_vaa`,
    arguments: [tx.object(WORMHOLE_STATE_ID), verifiedVaa, decreeTicket],
    typeArguments: [
      `${newTokenBridgePackage}::upgrade_contract::GovernanceWitness`,
    ],
  });
  tx.moveCall({
    target: `${newTokenBridgePackage}::migrate::migrate`,
    arguments: [tx.object(TOKEN_BRIDGE_STATE_ID), decreeReceipt],
  });

  return tx;
}

main().catch((e) => {
  console.error(e);
  process.exit(1);
});
