import {
  RawSigner,
  SUI_CLOCK_OBJECT_ID,
  TransactionBlock,
} from "@mysten/sui.js";
import { buildForBytecode } from "./build";
import { getPackageId } from "./utils";

export async function buildAndUpgradeWormhole(
  signer: RawSigner,
  signedVaa: Buffer,
  wormholePath: string,
  wormholeStateId: string
) {
  const wormholePackage = await getPackageId(signer.provider, wormholeStateId);

  const tx = new TransactionBlock();

  // Authorize upgrade.
  const [upgradeTicket] = tx.moveCall({
    target: `${wormholePackage}::upgrade_contract::authorize_upgrade`,
    arguments: [
      tx.object(wormholeStateId),
      tx.pure(Array.from(signedVaa)),
      tx.object(SUI_CLOCK_OBJECT_ID),
    ],
  });

  // Build and generate modules and dependencies for upgrade.
  const { modules, dependencies } = buildForBytecode(wormholePath);
  const [upgradeReceipt] = tx.upgrade({
    modules,
    dependencies,
    packageId: wormholePackage,
    ticket: upgradeTicket,
  });

  // Commit upgrade.
  tx.moveCall({
    target: `${wormholePackage}::upgrade_contract::commit_upgrade`,
    arguments: [tx.object(wormholeStateId), upgradeReceipt],
  });

  // Cannot auto compute gas budget, so we need to configure it manually.
  // Gas ~215m.
  tx.setGasBudget(215_000_000n);

  return signer.signAndExecuteTransactionBlock({
    transactionBlock: tx,
    options: {
      showEffects: true,
      showEvents: true,
    },
  });
}

export async function migrate(signer: RawSigner, stateId: string) {
  const contractPackage = await getPackageId(signer.provider, stateId);

  const tx = new TransactionBlock();
  tx.moveCall({
    target: `${contractPackage}::migrate::migrate`,
    arguments: [tx.object(stateId)],
  });

  return signer.signAndExecuteTransactionBlock({
    transactionBlock: tx,
    options: {
      showEffects: true,
      showEvents: true,
    },
  });
}
