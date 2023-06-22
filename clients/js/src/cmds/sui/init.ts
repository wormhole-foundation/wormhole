import { SuiTransactionBlockResponse, TransactionBlock } from "@mysten/sui.js";
import yargs from "yargs";
import {
  executeTransactionBlock,
  getCreatedObjects,
  getOwnedObjectId,
  getPackageId,
  getProvider,
  getSigner,
  getUpgradeCapObjectId,
  isSameType,
  logTransactionDigest,
  logTransactionSender,
  setMaxGasBudgetDevnet,
} from "../../chains/sui";
import {
  DEBUG_OPTIONS,
  GOVERNANCE_CHAIN,
  GOVERNANCE_EMITTER,
  NETWORKS,
  NETWORK_OPTIONS,
  PRIVATE_KEY_OPTIONS,
  RPC_OPTIONS,
} from "../../consts";
import { Network, assertNetwork } from "../../utils";
import { YargsAddCommandsFn } from "../Yargs";

export const addInitCommands: YargsAddCommandsFn = (y: typeof yargs) =>
  y
    .command(
      "init-example-message-app",
      "Initialize example core message app",
      (yargs) =>
        yargs
          .option("network", NETWORK_OPTIONS)
          .option("package-id", {
            alias: "p",
            describe: "Example app package ID",
            demandOption: true,
            type: "string",
          })
          .option("wormhole-state", {
            alias: "w",
            describe: "Wormhole state object ID",
            demandOption: true,
            type: "string",
          })
          .option("private-key", PRIVATE_KEY_OPTIONS)
          .option("rpc", RPC_OPTIONS),
      async (argv) => {
        const network = argv.network.toUpperCase();
        assertNetwork(network);
        const packageId = argv["package-id"];
        const wormholeStateObjectId = argv["wormhole-state"];
        const privateKey = argv["private-key"];
        const rpc = argv.rpc;

        const res = await initExampleApp(
          network,
          packageId,
          wormholeStateObjectId,
          rpc,
          privateKey
        );

        logTransactionDigest(res);
        logTransactionSender(res);
        console.log(
          "Example app state object ID",
          getCreatedObjects(res).find((e) =>
            isSameType(e.type, `${packageId}::sender::State`)
          )?.objectId
        );
      }
    )
    .command(
      "init-token-bridge",
      "Initialize token bridge contract",
      (yargs) =>
        yargs
          .option("network", NETWORK_OPTIONS)
          .option("package-id", {
            alias: "p",
            describe: "Token bridge package ID",
            demandOption: true,
            type: "string",
          })
          .option("wormhole-state", {
            alias: "w",
            describe: "Wormhole state object ID",
            demandOption: true,
            type: "string",
          })
          .option("governance-chain-id", {
            alias: "c",
            describe: "Governance chain ID",
            default: GOVERNANCE_CHAIN,
            type: "number",
            demandOption: false,
          })
          .option("governance-address", {
            alias: "a",
            describe: "Governance contract address",
            type: "string",
            default: GOVERNANCE_EMITTER,
            demandOption: false,
          })
          .option("private-key", PRIVATE_KEY_OPTIONS)
          .option("rpc", RPC_OPTIONS),
      async (argv) => {
        const network = argv.network.toUpperCase();
        assertNetwork(network);
        const packageId = argv["package-id"];
        const wormholeStateObjectId = argv["wormhole-state"];
        const governanceChainId = argv["governance-chain-id"];
        const governanceContract = argv["governance-address"];
        const privateKey = argv["private-key"];
        const rpc = argv.rpc ?? NETWORKS[network].sui.rpc;

        const res = await initTokenBridge(
          network,
          packageId,
          wormholeStateObjectId,
          governanceChainId,
          governanceContract,
          rpc,
          privateKey
        );

        logTransactionDigest(res);
        logTransactionSender(res);
        console.log(
          "Token bridge state object ID",
          getCreatedObjects(res).find((e) =>
            isSameType(e.type, `${packageId}::state::State`)
          )?.objectId
        );
      }
    )
    .command(
      "init-wormhole",
      "Initialize wormhole core contract",
      (yargs) =>
        yargs
          .option("network", NETWORK_OPTIONS)
          .option("package-id", {
            alias: "p",
            describe: "Core bridge package ID",
            demandOption: true,
            type: "string",
          })
          .option("initial-guardian", {
            alias: "i",
            demandOption: true,
            describe: "Initial guardian public keys",
            type: "string",
          })
          .option("debug", DEBUG_OPTIONS)
          .option("governance-chain-id", {
            alias: "c",
            describe: "Governance chain ID",
            default: GOVERNANCE_CHAIN,
            type: "number",
            demandOption: false,
          })
          .option("guardian-set-index", {
            alias: "s",
            describe: "Governance set index",
            default: 0,
            type: "number",
            demandOption: false,
          })
          .option("governance-address", {
            alias: "a",
            describe: "Governance contract address",
            type: "string",
            default: GOVERNANCE_EMITTER,
            demandOption: false,
          })
          .option("private-key", PRIVATE_KEY_OPTIONS)
          .option("rpc", RPC_OPTIONS),
      async (argv) => {
        const network = argv.network.toUpperCase();
        assertNetwork(network);
        const packageId = argv["package-id"];
        const initialGuardian = argv["initial-guardian"];
        const debug = argv.debug ?? false;
        const governanceChainId = argv["governance-chain-id"];
        const guardianSetIndex = argv["guardian-set-index"];
        const governanceContract = argv["governance-address"];
        const privateKey = argv["private-key"];
        const rpc = argv.rpc;

        const res = await initWormhole(
          network,
          packageId,
          initialGuardian,
          governanceChainId,
          guardianSetIndex,
          governanceContract,
          rpc,
          privateKey
        );

        logTransactionDigest(res);
        console.log(
          "Wormhole state object ID",
          getCreatedObjects(res).find((e) =>
            isSameType(e.type, `${packageId}::state::State`)
          )?.objectId
        );
        if (debug) {
          logTransactionSender(res);
        }
      }
    );

export const initExampleApp = async (
  network: Network,
  packageId: string,
  wormholeStateObjectId: string,
  rpc?: string,
  privateKey?: string
): Promise<SuiTransactionBlockResponse> => {
  rpc = rpc ?? NETWORKS[network].sui.rpc;
  const provider = getProvider(network, rpc);
  const signer = getSigner(provider, network, privateKey);

  const tx = new TransactionBlock();
  setMaxGasBudgetDevnet(network, tx);
  tx.moveCall({
    target: `${packageId}::sender::init_with_params`,
    arguments: [tx.object(wormholeStateObjectId)],
  });
  return executeTransactionBlock(signer, tx);
};

export const initTokenBridge = async (
  network: Network,
  tokenBridgePackageId: string,
  coreBridgeStateObjectId: string,
  governanceChainId: number,
  governanceContract: string,
  rpc?: string,
  privateKey?: string
): Promise<SuiTransactionBlockResponse> => {
  rpc = rpc ?? NETWORKS[network].sui.rpc;
  const provider = getProvider(network, rpc);
  const signer = getSigner(provider, network, privateKey);
  const owner = await signer.getAddress();

  const deployerCapObjectId = await getOwnedObjectId(
    provider,
    owner,
    tokenBridgePackageId,
    "setup",
    "DeployerCap"
  );
  if (!deployerCapObjectId) {
    throw new Error(
      `Token bridge cannot be initialized because deployer capability cannot be found under ${owner}. Is the package published?`
    );
  }

  const upgradeCapObjectId = await getUpgradeCapObjectId(
    provider,
    owner,
    tokenBridgePackageId
  );
  if (!upgradeCapObjectId) {
    throw new Error(
      `Token bridge cannot be initialized because upgrade capability cannot be found under ${owner}. Is the package published?`
    );
  }

  const wormholePackageId = await getPackageId(
    provider,
    coreBridgeStateObjectId
  );

  const tx = new TransactionBlock();
  setMaxGasBudgetDevnet(network, tx);
  const [emitterCap] = tx.moveCall({
    target: `${wormholePackageId}::emitter::new`,
    arguments: [tx.object(coreBridgeStateObjectId)],
  });
  tx.moveCall({
    target: `${tokenBridgePackageId}::setup::complete`,
    arguments: [
      tx.object(deployerCapObjectId),
      tx.object(upgradeCapObjectId),
      emitterCap,
      tx.pure(governanceChainId),
      tx.pure([...Buffer.from(governanceContract, "hex")]),
    ],
  });
  return executeTransactionBlock(signer, tx);
};

export const initWormhole = async (
  network: Network,
  coreBridgePackageId: string,
  initialGuardians: string,
  governanceChainId: number,
  guardianSetIndex: number,
  governanceContract: string,
  rpc?: string,
  privateKey?: string
): Promise<SuiTransactionBlockResponse> => {
  rpc = rpc ?? NETWORKS[network].sui.rpc;
  const provider = getProvider(network, rpc);
  const signer = getSigner(provider, network, privateKey);
  const owner = await signer.getAddress();

  const deployerCapObjectId = await getOwnedObjectId(
    provider,
    owner,
    coreBridgePackageId,
    "setup",
    "DeployerCap"
  );
  if (!deployerCapObjectId) {
    throw new Error(
      `Wormhole cannot be initialized because deployer capability cannot be found under ${owner}. Is the package published?`
    );
  }

  const upgradeCapObjectId = await getUpgradeCapObjectId(
    provider,
    owner,
    coreBridgePackageId
  );
  if (!upgradeCapObjectId) {
    throw new Error(
      `Wormhole cannot be initialized because upgrade capability cannot be found under ${owner}. Is the package published?`
    );
  }

  const tx = new TransactionBlock();
  setMaxGasBudgetDevnet(network, tx);
  tx.moveCall({
    target: `${coreBridgePackageId}::setup::complete`,
    arguments: [
      tx.object(deployerCapObjectId),
      tx.object(upgradeCapObjectId),
      tx.pure(governanceChainId),
      tx.pure([...Buffer.from(governanceContract, "hex")]),
      tx.pure(guardianSetIndex),
      tx.pure(
        initialGuardians.split(",").map((g) => [...Buffer.from(g, "hex")])
      ),
      tx.pure(24 * 60 * 60), // Guardian set TTL in seconds
      tx.pure("0"), // Message fee
    ],
  });
  return executeTransactionBlock(signer, tx);
};
