import { SuiTransactionBlockResponse, TransactionBlock } from "@mysten/sui.js";
import yargs from "yargs";
import {
  DEBUG_OPTIONS,
  GOVERNANCE_CHAIN,
  GOVERNANCE_EMITTER,
  NETWORK_OPTIONS,
  PRIVATE_KEY_OPTIONS,
  RPC_OPTIONS,
} from "../../consts";
import { NETWORKS } from "../../networks";
import {
  executeTransactionBlock,
  getCreatedObjects,
  getOwnedObjectId,
  getProvider,
  getSigner,
  getUpgradeCapObjectId,
  isSameType,
} from "../../sui";
import { logTransactionDigest, logTransactionSender } from "../../sui/log";
import { Network, assertNetwork } from "../../utils";
import { YargsAddCommandsFn } from "../Yargs";

export const addInitCommands: YargsAddCommandsFn = (y: typeof yargs) =>
  y
    .command(
      "init-example-message-app",
      "Initialize example core message app",
      (yargs) => {
        return yargs
          .option("network", NETWORK_OPTIONS)
          .option("package-id", {
            alias: "p",
            describe: "Package ID/module address",
            required: true,
            type: "string",
          })
          .option("wormhole-state", {
            alias: "w",
            describe: "Wormhole state object ID",
            required: true,
            type: "string",
          })
          .option("private-key", PRIVATE_KEY_OPTIONS)
          .option("rpc", RPC_OPTIONS);
      },
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
          ).objectId
        );
      }
    )
    .command(
      "init-token-bridge",
      "Initialize token bridge contract",
      (yargs) => {
        return yargs
          .option("network", NETWORK_OPTIONS)
          .option("package-id", {
            alias: "p",
            describe: "Package ID/module address",
            required: true,
            type: "string",
          })
          .option("wormhole-package-id", {
            alias: "i",
            describe: "Wormhole package ID/module address",
            required: true,
            type: "string",
          })
          .option("wormhole-state", {
            alias: "w",
            describe: "Wormhole state object ID",
            required: true,
            type: "string",
          })
          .option("governance-chain-id", {
            alias: "c",
            describe: "Governance chain ID",
            default: GOVERNANCE_CHAIN,
            type: "number",
            required: false,
          })
          .option("governance-address", {
            alias: "a",
            describe: "Governance contract address",
            type: "string",
            default: GOVERNANCE_EMITTER,
            required: false,
          })
          .option("private-key", PRIVATE_KEY_OPTIONS)
          .option("rpc", RPC_OPTIONS);
      },
      async (argv) => {
        const network = argv.network.toUpperCase();
        assertNetwork(network);
        const packageId = argv["package-id"];
        const wormholePackageId = argv["wormhole-package-id"];
        const wormholeStateObjectId = argv["wormhole-state"];
        const governanceChainId = argv["governance-chain-id"];
        const governanceContract = argv["governance-address"];
        const privateKey = argv["private-key"];
        const rpc = argv.rpc;

        const res = await initTokenBridge(
          network,
          packageId,
          wormholeStateObjectId,
          rpc,
          privateKey
        );

        logTransactionDigest(res);
        logTransactionSender(res);
        console.log(
          "Token bridge state object ID",
          getCreatedObjects(res).find((e) =>
            isSameType(e.type, `${packageId}::state::State`)
          ).objectId
        );
      }
    )
    .command(
      "init-wormhole",
      "Initialize wormhole core contract",
      (yargs) => {
        return yargs
          .option("network", NETWORK_OPTIONS)
          .option("package-id", {
            alias: "p",
            describe: "Package ID/module address",
            required: true,
            type: "string",
          })
          .option("initial-guardian", {
            alias: "i",
            required: true,
            describe: "Initial guardian public keys",
            type: "string",
          })
          .option("debug", DEBUG_OPTIONS)
          .option("governance-chain-id", {
            alias: "c",
            describe: "Governance chain ID",
            default: GOVERNANCE_CHAIN,
            type: "number",
            required: false,
          })
          .option("governance-address", {
            alias: "a",
            describe: "Governance contract address",
            type: "string",
            default: GOVERNANCE_EMITTER,
            required: false,
          })
          .option("private-key", PRIVATE_KEY_OPTIONS)
          .option("rpc", RPC_OPTIONS);
      },
      async (argv) => {
        const network = argv.network.toUpperCase();
        assertNetwork(network);
        const packageId = argv["package-id"];
        const initialGuardian = argv["initial-guardian"];
        const debug = argv.debug ?? false;
        const governanceChainId = argv["governance-chain-id"];
        const governanceContract = argv["governance-address"];
        const privateKey = argv["private-key"];
        const rpc = argv.rpc;

        const res = await initWormhole(
          network,
          packageId,
          initialGuardian,
          governanceChainId,
          governanceContract,
          rpc,
          privateKey
        );

        logTransactionDigest(res);
        console.log(
          "Wormhole state object ID",
          getCreatedObjects(res).find((e) =>
            isSameType(e.type, `${packageId}::state::State`)
          ).objectId
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

  const transactionBlock = new TransactionBlock();
  transactionBlock.moveCall({
    target: `${packageId}::sender::init_with_params`,
    arguments: [transactionBlock.object(wormholeStateObjectId)],
  });
  return executeTransactionBlock(signer, transactionBlock);
};

export const initTokenBridge = async (
  network: Network,
  packageId: string,
  wormholeStateObjectId: string,
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
    packageId,
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
    packageId
  );
  if (!upgradeCapObjectId) {
    throw new Error(
      `Token bridge cannot be initialized because upgrade capability cannot be found under ${owner}. Is the package published?`
    );
  }

  const transactionBlock = new TransactionBlock();
  transactionBlock.moveCall({
    target: `${packageId}::setup::complete`,
    arguments: [
      transactionBlock.object(wormholeStateObjectId),
      transactionBlock.object(deployerCapObjectId),
      transactionBlock.object(upgradeCapObjectId),
    ],
  });
  return executeTransactionBlock(signer, transactionBlock);
};

export const initWormhole = async (
  network: Network,
  packageId: string,
  initialGuardian: string,
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
    packageId,
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
    packageId
  );
  if (!upgradeCapObjectId) {
    throw new Error(
      `Wormhole cannot be initialized because upgrade capability cannot be found under ${owner}. Is the package published?`
    );
  }

  const transactionBlock = new TransactionBlock();
  transactionBlock.moveCall({
    target: `${packageId}::setup::complete`,
    arguments: [
      transactionBlock.object(deployerCapObjectId),
      transactionBlock.object(upgradeCapObjectId),
      transactionBlock.pure(governanceChainId),
      transactionBlock.pure([...Buffer.from(governanceContract, "hex")]),
      transactionBlock.pure([[...Buffer.from(initialGuardian, "hex")]]),
      transactionBlock.pure(365 * 24 * 60 * 60), // Guardian set TTL in seconds
      transactionBlock.pure("0"), // Message fee
    ],
  });
  return executeTransactionBlock(signer, transactionBlock);
};
