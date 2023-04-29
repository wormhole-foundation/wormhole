import * as mock from "@certusone/wormhole-sdk/lib/cjs/mock";
import {
  RawSigner,
  SUI_CLOCK_OBJECT_ID,
  TransactionBlock,
  fromB64,
  normalizeSuiObjectId,
  JsonRpcProvider,
  Ed25519Keypair,
  testnetConnection,
} from "@mysten/sui.js";
import { execSync } from "child_process";
import { resolve } from "path";
import * as fs from "fs";

const GOVERNANCE_EMITTER =
  "0000000000000000000000000000000000000000000000000000000000000004";

const TOKEN_BRIDGE_STATE_ID =
  "0x32422cb2f929b6a4e3f81b4791ea11ac2af896b310f3d9442aa1fe924ce0bab4";
const WORMHOLE_STATE_ID =
  "0x69ae41bdef4770895eb4e7aaefee5e4673acc08f6917b4856cf55549c4573ca8";

async function main() {
  const guardianPrivateKey = process.env.TESTNET_GUARDIAN_PRIVATE_KEY;
  if (guardianPrivateKey === undefined) {
    throw new Error("TESTNET_GUARDIAN_PRIVATE_KEY unset in environment");
  }

  const walletPrivateKey = process.env.TESTNET_WALLET_PRIVATE_KEY;
  if (walletPrivateKey === undefined) {
    throw new Error("TESTNET_WALLET_PRIVATE_KEY unset in environment");
  }

  const provider = new JsonRpcProvider(testnetConnection);
  const wallet = new RawSigner(
    Ed25519Keypair.fromSecretKey(
      Buffer.from(walletPrivateKey, "base64").subarray(1)
    ),
    provider
  );

  const dstTokenBridgePath = resolve(`${__dirname}/../../token_bridge`);

  // Build for digest.
  const { modules, dependencies, digest } =
    buildForBytecodeAndDigest(dstTokenBridgePath);
  console.log("dependencies", dependencies);
  console.log("digest", digest.toString("hex"));

  // We will use the signed VAA when we execute the upgrade.
  const guardians = new mock.MockGuardians(0, [guardianPrivateKey]);

  const timestamp = 12345678;
  const governance = new mock.GovernanceEmitter(GOVERNANCE_EMITTER);
  const published = governance.publishWormholeUpgradeContract(
    timestamp,
    2,
    "0x" + digest.toString("hex")
  );
  const moduleName = Buffer.alloc(32);
  moduleName.write("TokenBridge", 32 - "TokenBridge".length);
  published.write(moduleName.toString(), 84 - 33);
  published.writeUInt16BE(21, 84);
  published.writeUInt8(2, 83);
  //message.writeUInt8(1, 83);
  published.writeUInt16BE(21, published.length - 34);

  const signedVaa = guardians.addSignatures(published, [0]);
  console.log("Upgrade VAA:", signedVaa.toString("hex"));

  // // And execute upgrade with governance VAA.
  // const upgradeResults = await upgradeTokenBridge(
  //   wallet,
  //   TOKEN_BRIDGE_STATE_ID,
  //   WORMHOLE_STATE_ID,
  //   modules,
  //   dependencies,
  //   signedVaa
  // );

  // console.log("tx digest", upgradeResults.digest);
  // console.log("tx effects", JSON.stringify(upgradeResults.effects!));
  // console.log("tx events", JSON.stringify(upgradeResults.events!));

  // TODO: grab new package ID from the events above. Do not rely on the RPC
  // call because it may give you a stale package ID after the upgrade.

  const migrateResults = await migrateTokenBridge(
    wallet,
    TOKEN_BRIDGE_STATE_ID,
    WORMHOLE_STATE_ID,
    signedVaa
  );
  console.log("tx digest", migrateResults.digest);
  console.log("tx effects", JSON.stringify(migrateResults.effects!));
  console.log("tx events", JSON.stringify(migrateResults.events!));
}

main();

// Yeah buddy.

function buildForBytecodeAndDigest(packagePath: string) {
  const buildOutput: {
    modules: string[];
    dependencies: string[];
    digest: number[];
  } = JSON.parse(
    execSync(
      `sui move build --dump-bytecode-as-base64 -p ${packagePath} 2> /dev/null`,
      { encoding: "utf-8" }
    )
  );
  return {
    modules: buildOutput.modules.map((m: string) => Array.from(fromB64(m))),
    dependencies: buildOutput.dependencies.map((d: string) =>
      normalizeSuiObjectId(d)
    ),
    digest: Buffer.from(buildOutput.digest),
  };
}

async function getPackageId(
  provider: JsonRpcProvider,
  stateId: string
): Promise<string> {
  const state = await provider
    .getObject({
      id: stateId,
      options: {
        showContent: true,
      },
    })
    .then((result) => {
      if (result.data?.content?.dataType == "moveObject") {
        return result.data.content.fields;
      }

      throw new Error("not move object");
    });

  if ("upgrade_cap" in state) {
    return state.upgrade_cap.fields.package;
  }

  throw new Error("upgrade_cap not found");
}

async function upgradeTokenBridge(
  signer: RawSigner,
  tokenBridgeStateId: string,
  wormholeStateId: string,
  modules: number[][],
  dependencies: string[],
  signedVaa: Buffer
) {
  const tokenBridgePackage = await getPackageId(
    signer.provider,
    tokenBridgeStateId
  );
  const wormholePackage = await getPackageId(signer.provider, wormholeStateId);

  const tx = new TransactionBlock();

  const [verifiedVaa] = tx.moveCall({
    target: `${wormholePackage}::vaa::parse_and_verify`,
    arguments: [
      tx.object(wormholeStateId),
      tx.pure(Array.from(signedVaa)),
      tx.object(SUI_CLOCK_OBJECT_ID),
    ],
  });
  const [decreeTicket] = tx.moveCall({
    target: `${tokenBridgePackage}::upgrade_contract::authorize_governance`,
    arguments: [tx.object(tokenBridgeStateId)],
  });
  const [decreeReceipt] = tx.moveCall({
    target: `${wormholePackage}::governance_message::verify_vaa`,
    arguments: [tx.object(wormholeStateId), verifiedVaa, decreeTicket],
    typeArguments: [
      `${tokenBridgePackage}::upgrade_contract::GovernanceWitness`,
    ],
  });

  // Authorize upgrade.
  const [upgradeTicket] = tx.moveCall({
    target: `${tokenBridgePackage}::upgrade_contract::authorize_upgrade`,
    arguments: [tx.object(tokenBridgeStateId), decreeReceipt],
  });

  // Build and generate modules and dependencies for upgrade.
  const [upgradeReceipt] = tx.upgrade({
    modules,
    dependencies,
    packageId: tokenBridgePackage,
    ticket: upgradeTicket,
  });

  // Commit upgrade.
  tx.moveCall({
    target: `${tokenBridgePackage}::upgrade_contract::commit_upgrade`,
    arguments: [tx.object(tokenBridgeStateId), upgradeReceipt],
  });

  // Cannot auto compute gas budget, so we need to configure it manually.
  // Gas ~215m.
  //tx.setGasBudget(1_000_000_000n);

  return signer.signAndExecuteTransactionBlock({
    transactionBlock: tx,
    options: {
      showEffects: true,
      showEvents: true,
    },
  });
}

async function migrateTokenBridge(
  signer: RawSigner,
  tokenBridgeStateId: string,
  wormholeStateId: string,
  signedUpgradeVaa: Buffer
) {
  const tokenBridgePackage = await getPackageId(
    signer.provider,
    tokenBridgeStateId
  );
  const wormholePackage = await getPackageId(signer.provider, wormholeStateId);

  const tx = new TransactionBlock();

  const [verifiedVaa] = tx.moveCall({
    target: `${wormholePackage}::vaa::parse_and_verify`,
    arguments: [
      tx.object(wormholeStateId),
      tx.pure(Array.from(signedUpgradeVaa)),
      tx.object(SUI_CLOCK_OBJECT_ID),
    ],
  });
  const [decreeTicket] = tx.moveCall({
    target: `${tokenBridgePackage}::upgrade_contract::authorize_governance`,
    arguments: [tx.object(tokenBridgeStateId)],
  });
  const [decreeReceipt] = tx.moveCall({
    target: `${wormholePackage}::governance_message::verify_vaa`,
    arguments: [tx.object(wormholeStateId), verifiedVaa, decreeTicket],
    typeArguments: [
      `${tokenBridgePackage}::upgrade_contract::GovernanceWitness`,
    ],
  });
  tx.moveCall({
    target: `${tokenBridgePackage}::migrate::migrate`,
    arguments: [tx.object(tokenBridgeStateId), decreeReceipt],
  });

  return signer.signAndExecuteTransactionBlock({
    transactionBlock: tx,
    options: {
      showEffects: true,
      showEvents: true,
    },
  });
}

function setUpWormholeDirectory(
  srcWormholePath: string,
  dstWormholePath: string
) {
  fs.cpSync(srcWormholePath, dstWormholePath, { recursive: true });

  // Remove irrelevant files. This part is not necessary, but is helpful
  // for debugging a clean package directory.
  const removeThese = [
    "Move.devnet.toml",
    "Move.lock",
    "Makefile",
    "README.md",
    "build",
  ];
  for (const basename of removeThese) {
    fs.rmSync(`${dstWormholePath}/${basename}`, {
      recursive: true,
      force: true,
    });
  }

  // Fix Move.toml file.
  const moveTomlPath = `${dstWormholePath}/Move.toml`;
  const moveToml = fs.readFileSync(moveTomlPath, "utf-8");
  fs.writeFileSync(
    moveTomlPath,
    moveToml.replace(`wormhole = "_"`, `wormhole = "0x0"`),
    "utf-8"
  );
}

function cleanUpPackageDirectory(packagePath: string) {
  fs.rmSync(packagePath, { recursive: true, force: true });
}
