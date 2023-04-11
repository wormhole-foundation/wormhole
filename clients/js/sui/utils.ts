import { CHAIN_ID_SUI } from "@certusone/wormhole-sdk";
import { CHAIN_ID_TO_NAME } from "@certusone/wormhole-sdk/lib/cjs/utils/consts";
import {
  Connection,
  Ed25519Keypair,
  fromB64,
  JsonRpcProvider,
  normalizeSuiAddress,
  RawSigner,
  SuiTransactionBlockResponse,
  SUI_CLOCK_OBJECT_ID,
  TransactionBlock,
} from "@mysten/sui.js";
import { CONTRACTS } from "../consts";
import { NETWORKS } from "../networks";
import { Network } from "../utils";
import { impossible, Payload } from "../vaa";
import { SuiAddresses, SUI_OBJECT_IDS } from "./consts";
import { SuiRpcValidationError } from "./error";
import { SuiCreateEvent, SuiPublishEvent } from "./types";

const UPGRADE_CAP_TYPE = "0x2::package::UpgradeCap";

export const execute_sui = async (
  payload: Payload,
  vaa: Buffer,
  network: Network,
  packageId?: string,
  addresses?: Partial<SuiAddresses>,
  rpc?: string,
  privateKey?: string
) => {
  const chain = CHAIN_ID_TO_NAME[CHAIN_ID_SUI];
  const provider = getProvider(network, rpc);
  const signer = getSigner(provider, network, privateKey);
  addresses = { ...SUI_OBJECT_IDS, ...addresses };

  switch (payload.module) {
    case "Core":
      packageId = packageId ?? CONTRACTS[network][chain]["core"];
      if (!packageId) {
        throw Error("Core bridge contract is undefined");
      }

      switch (payload.type) {
        case "GuardianSetUpgrade": {
          console.log("Submitting new guardian set");
          const tx = new TransactionBlock();
          tx.moveCall({
            target: `${packageId}::wormhole::update_guardian_set`,
            arguments: [
              tx.object(addresses[network].core_state),
              tx.pure([...vaa]),
              tx.object(SUI_CLOCK_OBJECT_ID),
            ],
          });
          await executeTransactionBlock(signer, tx);
          break;
        }
        case "ContractUpgrade":
          throw new Error("ContractUpgrade not supported on Sui");
        case "RecoverChainId":
          throw new Error("RecoverChainId not supported on Sui");
        default:
          impossible(payload);
      }
      break;
    case "NFTBridge":
      throw new Error("NFT bridge not supported on Sui");
    case "TokenBridge":
      packageId = packageId ?? CONTRACTS[network][chain]["token_bridge"];
      if (!packageId) {
        throw Error("Token bridge contract is undefined");
      }

      switch (payload.type) {
        case "ContractUpgrade":
          throw new Error("ContractUpgrade not supported on Sui");
        case "RecoverChainId":
          throw new Error("RecoverChainId not supported on Sui");
        case "RegisterChain": {
          console.log("Registering chain");
          const tx = new TransactionBlock();
          tx.setGasBudget(1000000);
          tx.moveCall({
            target: `${packageId}::register_chain::register_chain`,
            arguments: [
              tx.object(addresses[network].token_bridge_state),
              tx.object(addresses[network].core_state),
              tx.pure([...vaa]),
              tx.object(SUI_CLOCK_OBJECT_ID),
            ],
          });
          await executeTransactionBlock(signer, tx);
          break;
        }
        case "AttestMeta":
          throw new Error("AttestMeta not supported on Sui");
        case "Transfer":
          throw new Error("Transfer not supported on Sui");
        case "TransferWithPayload":
          throw Error("Can't complete payload 3 transfer from CLI");
        default:
          impossible(payload);
          break;
      }
      break;
    default:
      impossible(payload);
  }
};

export const executeTransactionBlock = async (
  signer: RawSigner,
  transactionBlock: TransactionBlock
): Promise<SuiTransactionBlockResponse> => {
  // Let caller handle parsing and logging info
  return signer.signAndExecuteTransactionBlock({
    transactionBlock,
    options: {
      showInput: true,
      showEffects: true,
      showEvents: true,
      showObjectChanges: true,
    },
  });
};

export const getCreatedObjects = (
  res: SuiTransactionBlockResponse
): { type: string; objectId: string; owner: string }[] => {
  return res.objectChanges.filter(isSuiCreateEvent).map((e) => ({
    type: e.objectType,
    objectId: e.objectId,
    owner: e.owner["AddressOwner"] || e.owner["ObjectOwner"] || e.owner,
  }));
};

export const getOwnedObjectId = async (
  provider: JsonRpcProvider,
  owner: string,
  packageId: string,
  moduleName: string,
  structName: string
): Promise<string | null> => {
  const type = `${packageId}::${moduleName}::${structName}`;

  // Upgrade caps are a special case
  if (normalizeSuiType(type) === normalizeSuiType(UPGRADE_CAP_TYPE)) {
    throw new Error(
      "`getOwnedObjectId` should not be used to get the object ID of an `UpgradeCap`. Use `getUpgradeCapObjectId` instead."
    );
  }

  const res = await provider.getOwnedObjects({
    owner,
    filter: { StructType: type },
    options: {
      showContent: true,
    },
  });
  if (!res || !res.data) {
    throw new SuiRpcValidationError(res);
  }

  const objects = res.data.filter((o) => o.data?.objectId);
  if (objects.length === 1) {
    return objects[0].data?.objectId;
  } else if (objects.length > 1) {
    const objectsStr = JSON.stringify(objects, null, 2);
    throw new Error(
      `Found multiple objects owned by ${owner} of type ${type}. This may mean that we've received an unexpected response from the Sui RPC and \`worm\` logic needs to be updated to handle this. Objects: ${objectsStr}`
    );
  } else {
    return null;
  }
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
  network: Network,
  customPrivateKey?: string
): RawSigner => {
  const privateKey: string | undefined =
    customPrivateKey || NETWORKS[network]["sui"].key;
  if (!privateKey) {
    throw new Error(`No private key found for Sui ${network}`);
  }

  const bytes = fromB64(privateKey);
  const keypair = Ed25519Keypair.fromSecretKey(bytes.slice(1));
  return new RawSigner(keypair, provider);
};

/**
 * This function returns the object ID of the `UpgradeCap` that belongs to the
 * given package and owner if it exists.
 *
 * Structs created by the Sui framework such as `UpgradeCap`s all have the same
 * type (e.g. `0x2::package::UpgradeCap`) and have a special field, `package`,
 * we can use to differentiate them.
 * @param provider Sui RPC provider
 * @param owner Address of the current owner of the `UpgradeCap`
 * @param packageId ID of the package that the `UpgradeCap` was created for
 * @returns The object ID of the `UpgradeCap` if it exists, otherwise `null`
 */
export const getUpgradeCapObjectId = async (
  provider: JsonRpcProvider,
  owner: string,
  packageId: string
): Promise<string | null> => {
  const res = await provider.getOwnedObjects({
    owner,
    filter: { StructType: UPGRADE_CAP_TYPE },
    options: {
      showContent: true,
    },
  });
  if (!res || !res.data) {
    throw new SuiRpcValidationError(res);
  }

  const objects = res.data.filter(
    (o) =>
      o.data?.objectId &&
      o.data?.content?.dataType === "moveObject" &&
      o.data?.content?.fields?.package === packageId
  );
  if (objects.length === 1) {
    // We've found the object we're looking for
    return objects[0].data?.objectId;
  } else if (objects.length > 1) {
    const objectsStr = JSON.stringify(objects, null, 2);
    throw new Error(
      `Found multiple upgrade capabilities owned by ${owner} from package ${packageId}. Objects: ${objectsStr}`
    );
  } else {
    return null;
  }
};

export const isSameType = (a: string, b: string) => {
  try {
    return normalizeSuiType(a) === normalizeSuiType(b);
  } catch (e) {
    return false;
  }
};

export const isSuiPublishEvent = (event: any): event is SuiPublishEvent => {
  return event.type === "published";
};

export const isSuiCreateEvent = (event: any): event is SuiCreateEvent => {
  return event.type === "created";
};

export const isValidSuiAddress = (objectId: string): boolean => {
  return /^(0x)?[0-9a-f]{1,64}$/.test(objectId);
};

// todo(aki): this needs to correctly handle types such as
// 0x2::dynamic_field::Field<0x3c6d386861470e6f9cb35f3c91f69e6c1f1737bd5d217ca06a15f582e1dc1ce3::state::MigrationControl, bool>
export const normalizeSuiType = (type: string): string => {
  const tokens = type.split("::");
  if (tokens.length !== 3 || !isValidSuiAddress(tokens[0])) {
    throw new Error(`Invalid Sui type: ${type}`);
  }

  return [normalizeSuiAddress(tokens[0]), tokens[1], tokens[2]].join("::");
};
