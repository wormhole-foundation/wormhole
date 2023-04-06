import {
  Connection,
  Ed25519Keypair,
  fromB64,
  JsonRpcProvider,
  RawSigner,
  TransactionBlock,
} from "@mysten/sui.js";
import { NETWORKS } from "../networks";
import { Network } from "../utils";
import { SuiCreateEvent, SuiPublishEvent } from "./types";

export async function executeTransactionBlock(
  signer: RawSigner,
  transactionBlock: TransactionBlock
) {
  const testRes = await signer.dryRunTransactionBlock({ transactionBlock });
  if (testRes.effects.status.status !== "success") {
    throw new Error(
      `Failed to execute transaction: ${testRes.effects.status.error}`
    );
  }

  const res = await signer.signAndExecuteTransactionBlock({
    transactionBlock,
    options: {
      showInput: true,
      showEffects: true,
      showEvents: true,
      showObjectChanges: true,
    },
  });

  console.log("Digest", res.digest, res.effects.transactionDigest);
  console.log("Sender", res.transaction.data.sender);

  // Let caller handle parsing and logging info
  return res;
}

export const getOwnedObjectId = async (
  provider: JsonRpcProvider,
  owner: string,
  packageId: string,
  moduleName: string,
  structName: string
): Promise<string | null> => {
  const type = `${packageId}::${moduleName}::${structName}`;
  const objects = (
    await provider.getOwnedObjects({
      owner,
      filter: { StructType: type },
      options: {
        showContent: true,
      },
    })
  ).data.filter((o) => o.data?.objectId);

  // Structs such as UpgradeCaps have the same type and have another field we
  // can use to differentiate them. We have to check this first as we could have
  // only one UpgradeCap that belongs to a different package.
  const filteredObjects = objects.filter(
    (o) =>
      o.data?.content?.dataType === "moveObject" &&
      o.data?.content?.fields?.package === packageId
  );
  if (filteredObjects.length === 1) {
    // We've found the object we're looking for
    return filteredObjects[0].data?.objectId;
  } else if (filteredObjects.length > 1) {
    const objectsStr = JSON.stringify(filteredObjects, null, 2);
    throw new Error(
      `Found multiple objects owned by ${owner} of type ${type}. Objects: ${objectsStr}`
    );
  }

  // Those properties aren't returned for other structs (as of Sui SDK ver.
  // 0.30.0) and we can assume that if we've found a single object with the
  // correct type, that's what we're looking for.
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

export const isValidSuiObjectId = (objectId: string): boolean => {
  return /^(0x)?[0-9a-f]{64}$/i.test(objectId);
};

export const isSuiPublishEvent = (event: any): event is SuiPublishEvent => {
  return event.type === "published";
};

export const isSuiCreateEvent = (event: any): event is SuiCreateEvent => {
  return event.type === "created";
};
