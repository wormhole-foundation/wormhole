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
  provider: JsonRpcProvider,
  network: Network,
  transactionBlock: TransactionBlock
) {
  const signer = getSigner(provider, network);
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
  const res = await provider.getOwnedObjects({
    owner,
    filter: { StructType: `${packageId}::${moduleName}::${structName}` },
  });
  return res.data.length > 0 ? res.data[0].data.objectId : null;
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
  network: Network
): RawSigner => {
  const privateKey: string | undefined = NETWORKS[network]["sui"].key;
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
