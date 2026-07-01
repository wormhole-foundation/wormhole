import { SuiGrpcClient } from "@mysten/sui/grpc";
import type { SuiClientTypes } from "@mysten/sui/client";
import { Ed25519Keypair } from "@mysten/sui/keypairs/ed25519";
import { Transaction } from "@mysten/sui/transactions";
import { fromBase64 } from "@mysten/sui/utils";
import { NETWORKS } from "../../consts";
import { Payload, VAA, parse, serialiseVAA } from "../../vaa";
import { Network } from "@wormhole-foundation/sdk";
import { isValidSuiAddress } from "../../sdk/sui";

const SUI_CLOCK_OBJECT_ID =
  "0x0000000000000000000000000000000000000000000000000000000000000006";
const UPGRADE_CAP_TYPE = "0x2::package::UpgradeCap";

// Re-export for other modules
export { SUI_CLOCK_OBJECT_ID };

// Type for signer - combines keypair, gRPC client, and the selected network
// (used to verify the endpoint matches before signing).
export interface SuiSigner {
  keypair: Ed25519Keypair;
  client: SuiGrpcClient;
  network: Network;
}

/**
 * Normalized view of an object touched by a transaction. The gRPC transport
 * exposes mutations as `effects.changedObjects` plus an `objectId -> type` map,
 * which we flatten here so callers don't need to reassemble the two.
 */
export interface SuiChangedObject {
  objectId: string;
  type?: string;
  owner: string;
  created: boolean;
  isPackage: boolean;
}

/**
 * Normalized transaction result. The gRPC client returns a discriminated
 * `{ Transaction | FailedTransaction }` envelope whose shape differs from the
 * legacy JSON-RPC `SuiTransactionBlockResponse`; this is the single shape the
 * rest of the CLI consumes.
 */
export interface SuiTransactionResult {
  digest: string;
  success: boolean;
  error?: string;
  sender?: string;
  changedObjects: SuiChangedObject[];
  events: {
    packageId: string;
    module: string;
    sender: string;
    eventType: string;
    json: Record<string, unknown> | null;
  }[];
}

export const getSuiNetwork = (
  network?: Network
): "mainnet" | "testnet" | "localnet" => {
  switch (network) {
    case "Mainnet":
      return "mainnet";
    case "Testnet":
      return "testnet";
    // Wormhole's "Devnet" is a local Sui node, which Sui itself labels
    // "localnet". An unset network (RPC supplied without --network) gets the
    // same conservative label, since the gRPC client uses `network` only as an
    // advisory hint and routes on `baseUrl`.
    case "Devnet":
    case undefined:
      return "localnet";
    default:
      throw new Error(`Unsupported Sui network: ${network}`);
  }
};

/**
 * Known Sui chain identifiers (the genesis checkpoint digest, fixed per chain)
 * used to ensure the selected `--network` and the possibly-overridden `--rpc`
 * endpoint actually agree before any transaction is signed. Devnet/localnet
 * identifiers are genesis-specific, so those networks are not asserted.
 */
const SUI_CHAIN_IDENTIFIERS: Partial<Record<Network, string>> = {
  Mainnet: "4btiuiMPvEENsttpZC7CZ53DruC3MAgfznDbASZ7DR6S",
  Testnet: "69WiPg3DAQiwdxfncX6wYQ2siKwAe6L9BZthQea3JNMD",
};

const verifiedClients = new WeakSet<SuiGrpcClient>();

/**
 * Guard against a network/endpoint mismatch (e.g. a Mainnet governance action
 * pointed at a Testnet or forked RPC). Fetches the endpoint's chain identifier
 * and throws if it does not match the expected `network`. No-op for networks
 * without a known identifier (Devnet/localnet). Result is cached per client.
 */
export const assertSuiNetwork = async (
  client: SuiGrpcClient,
  network: Network
): Promise<void> => {
  const expected = SUI_CHAIN_IDENTIFIERS[network];
  if (!expected || verifiedClients.has(client)) {
    return;
  }
  const { chainIdentifier } = await client.core.getChainIdentifier();
  if (chainIdentifier !== expected) {
    throw new Error(
      `Refusing to proceed: the Sui RPC reports chain identifier ${chainIdentifier}, ` +
        `but ${network} is expected to be ${expected}. ` +
        `Check that --rpc matches --network ${network}.`
    );
  }
  verifiedClients.add(client);
};

const ownerToString = (owner: {
  $kind: string;
  AddressOwner?: string;
  ObjectOwner?: string;
}): string => {
  if (owner.$kind === "AddressOwner" && owner.AddressOwner) {
    return owner.AddressOwner;
  }
  if (owner.$kind === "ObjectOwner" && owner.ObjectOwner) {
    return owner.ObjectOwner;
  }
  if (owner.$kind === "Shared") return "Shared";
  if (owner.$kind === "Immutable") return "Immutable";
  return "Unknown";
};

export const assertSuccess = (
  res: SuiTransactionResult,
  error: string
): void => {
  if (!res.success) {
    throw new Error(`${error} Response: ${JSON.stringify(res)}`);
  }
};

// Fields requested from the gRPC transport whenever a transaction result is
// flattened into a `SuiTransactionResult`. `objectTypes` is required to populate
// the per-object `type`, which the effects' `changedObjects` do not carry.
const TX_RESULT_INCLUDE = {
  effects: true,
  events: true,
  objectTypes: true,
  transaction: true,
} as const;

// The gRPC transaction envelope (the `Transaction`/`FailedTransaction` arm of a
// `TransactionResult`), parameterized by the fields we request so the
// include-gated members (`effects`, `objectTypes`, ...) are present on the type.
type GrpcTransaction = SuiClientTypes.Transaction<typeof TX_RESULT_INCLUDE>;

/**
 * Flatten a gRPC transaction envelope into the CLI's `SuiTransactionResult`.
 * The gRPC transport exposes mutations as `effects.changedObjects` (with
 * `outputState`/`idOperation` enums) plus a separate `objectId -> type` map;
 * this reassembles the two into the single shape the rest of the CLI consumes.
 */
export const toSuiTransactionResult = (
  tx: GrpcTransaction
): SuiTransactionResult => {
  const objectTypes = tx.objectTypes ?? {};
  return {
    digest: tx.digest,
    success: tx.status.success,
    error: tx.status.success
      ? undefined
      : tx.status.error?.message ?? JSON.stringify(tx.status.error),
    sender: tx.transaction?.sender ?? undefined,
    changedObjects: (tx.effects?.changedObjects ?? []).map((o) => ({
      objectId: o.objectId,
      type: objectTypes[o.objectId],
      owner: o.outputOwner ? ownerToString(o.outputOwner) : "Unknown",
      created: o.idOperation === "Created",
      isPackage: o.outputState === "PackageWrite",
    })),
    events: (tx.events ?? []).map((e) => ({
      packageId: e.packageId,
      module: e.module,
      sender: e.sender,
      eventType: e.eventType,
      json: e.json,
    })),
  };
};

export const executeTransactionBlock = async (
  signer: SuiSigner,
  transaction: Transaction
): Promise<SuiTransactionResult> => {
  await assertSuiNetwork(signer.client, signer.network);
  const res = await signer.client.signAndExecuteTransaction({
    signer: signer.keypair,
    transaction,
    include: TX_RESULT_INCLUDE,
  });

  const tx = res.Transaction ?? res.FailedTransaction;
  if (!tx) {
    throw new Error(
      `Unexpected empty transaction response: ${JSON.stringify(res)}`
    );
  }

  return toSuiTransactionResult(tx);
};

/**
 * Read an already-committed transaction's effects over gRPC and flatten them
 * into a `SuiTransactionResult`. Used for transactions executed outside the SDK
 * (e.g. `sui client test-publish`, which signs with the local CLI keystore and
 * commits on-chain) so their results can be consumed without parsing the CLI's
 * deprecated JSON-RPC `objectChanges` payload.
 */
export const fetchTransactionResult = async (
  client: SuiGrpcClient,
  digest: string
): Promise<SuiTransactionResult> => {
  // The transaction may not be queryable the instant the CLI returns; block
  // until the node has indexed it before reading effects.
  await client.waitForTransaction({ digest });
  const res = await client.getTransaction({
    digest,
    include: TX_RESULT_INCLUDE,
  });

  const tx = res.Transaction ?? res.FailedTransaction;
  if (!tx) {
    throw new Error(`Transaction ${digest} not found`);
  }

  return toSuiTransactionResult(tx);
};

export const getCreatedObjects = (
  res: SuiTransactionResult
): { type: string; objectId: string; owner: string }[] =>
  res.changedObjects
    .filter((o) => o.created && !o.isPackage && o.type)
    .map((o) => ({ type: o.type as string, objectId: o.objectId, owner: o.owner }));

export const getOwnedObjectId = async (
  client: SuiGrpcClient,
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

  const objectIds: string[] = [];
  let cursor: string | null = null;
  let hasNextPage = true;
  while (hasNextPage) {
    const res: SuiClientTypes.ListOwnedObjectsResponse =
      await client.listOwnedObjects({ owner, type, cursor });
    for (const o of res.objects) {
      if (o.type === type) {
        objectIds.push(o.objectId);
      }
    }
    hasNextPage = res.hasNextPage;
    cursor = res.cursor;
  }

  if (objectIds.length === 1) {
    return objectIds[0];
  } else if (objectIds.length > 1) {
    throw new Error(
      `Found multiple objects owned by ${owner} of type ${type}. This may mean that we've received an unexpected response from the Sui RPC and \`worm\` logic needs to be updated to handle this. Objects: ${JSON.stringify(
        objectIds,
        null,
        2
      )}`
    );
  }
  return null;
};

// TODO(kp): remove this once it's in the sdk
export const getPackageId = async (
  client: SuiGrpcClient,
  objectId: string
): Promise<string> => {
  let cursor: string | null = null;
  let hasNextPage = true;
  while (hasNextPage) {
    const res: SuiClientTypes.ListDynamicFieldsResponse =
      await client.listDynamicFields({ parentId: objectId, cursor });
    const currentPackage = res.dynamicFields.find((field) =>
      field.name.type.endsWith("CurrentPackage")
    );
    if (currentPackage) {
      const obj = await client.getObject({
        objectId: currentPackage.fieldId,
        include: { json: true },
      });
      const packageId = (obj.object.json as any)?.value?.package;
      if (!packageId) {
        throw new Error("Unable to get current package");
      }
      return packageId;
    }
    hasNextPage = res.hasNextPage;
    cursor = res.cursor;
  }

  throw new Error("CurrentPackage not found");
};

/**
 * Returns the original (type-origin) package ID for a state object. Move type
 * origins are stable across upgrades, so this differs from `getPackageId`, which
 * returns the current/upgraded package.
 */
export const getOriginalPackageId = async (
  client: SuiGrpcClient,
  stateObjectId: string
): Promise<string> => {
  const res = await client.getObject({ objectId: stateObjectId });
  return res.object.type.split("::")[0];
};

export const getProvider = (
  network?: Network,
  rpc?: string
): SuiGrpcClient => {
  if (!network && !rpc) {
    throw new Error("Must provide network or RPC to initialize provider");
  }

  rpc = rpc || NETWORKS[network!].Sui.rpc;
  if (!rpc) {
    throw new Error(`No default RPC found for Sui ${network}`);
  }

  return new SuiGrpcClient({ network: getSuiNetwork(network), baseUrl: rpc });
};

export const getPublishedPackageId = (res: SuiTransactionResult): string => {
  const packages = res.changedObjects.filter((o) => o.isPackage);
  if (packages.length !== 1) {
    throw new Error(
      "Unexpected number of published packages found:" +
        JSON.stringify(packages, null, 2)
    );
  }

  return packages[0].objectId;
};

export const getSigner = (
  client: SuiGrpcClient,
  network: Network,
  customPrivateKey?: string
): SuiSigner => {
  const privateKey: string | undefined =
    customPrivateKey || NETWORKS[network].Sui.key;
  if (!privateKey) {
    throw new Error(`No private key found for Sui ${network}`);
  }

  let bytes = privateKey.startsWith("0x")
    ? Buffer.from(privateKey.slice(2), "hex")
    : fromBase64(privateKey);
  if (bytes.length === 33) {
    // remove the first flag byte after checking it is indeed the Ed25519 scheme flag 0x00
    if (bytes[0] !== 0) {
      throw new Error("Only the Ed25519 scheme flag is supported");
    }
    bytes = bytes.subarray(1);
  }
  const keypair = Ed25519Keypair.fromSecretKey(Uint8Array.from(bytes));
  return { keypair, client, network };
};

/**
 * This function returns the object ID of the `UpgradeCap` that belongs to the
 * given package and owner if it exists.
 *
 * Structs created by the Sui framework such as `UpgradeCap`s all have the same
 * type (e.g. `0x2::package::UpgradeCap`) and have a special field, `package`,
 * we can use to differentiate them.
 * @param client Sui gRPC client
 * @param owner Address of the current owner of the `UpgradeCap`
 * @param packageId ID of the package that the `UpgradeCap` was created for
 * @returns The object ID of the `UpgradeCap` if it exists, otherwise `null`
 */
export const getUpgradeCapObjectId = async (
  client: SuiGrpcClient,
  owner: string,
  packageId: string
): Promise<string | null> => {
  const objectIds: string[] = [];
  let cursor: string | null = null;
  let hasNextPage = true;
  while (hasNextPage) {
    const res: SuiClientTypes.ListOwnedObjectsResponse<{ json: true }> =
      await client.listOwnedObjects({
        owner,
        type: UPGRADE_CAP_TYPE,
        cursor,
        include: { json: true },
      });
    for (const o of res.objects) {
      const pkg = (o.json as any)?.package;
      if (pkg === packageId) {
        objectIds.push(o.objectId);
      }
    }
    hasNextPage = res.hasNextPage;
    cursor = res.cursor;
  }

  if (objectIds.length === 1) {
    return objectIds[0];
  } else if (objectIds.length > 1) {
    throw new Error(
      `Found multiple upgrade capabilities owned by ${owner} from package ${packageId}. Objects: ${JSON.stringify(
        objectIds,
        null,
        2
      )}`
    );
  }
  return null;
};

export const isSameType = (a: string, b: string) => {
  try {
    return normalizeSuiType(a) === normalizeSuiType(b);
  } catch (e) {
    return false;
  }
};

// Normalize Sui address
export const normalizeSuiAddress = (address: string): string => {
  // Remove 0x prefix, pad to 64 chars, add 0x back
  const hex = address.replace(/^0x/, "").padStart(64, "0");
  return `0x${hex}`;
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

export const registerChain = async (
  client: SuiGrpcClient,
  network: Network,
  vaa: Buffer,
  coreBridgeStateObjectId: string,
  tokenBridgeStateObjectId: string,
  transaction?: Transaction
): Promise<Transaction> => {
  if (network === "Devnet") {
    // Modify the VAA to only have 1 guardian signature
    // TODO: remove this when we can deploy the devnet core contract
    // deterministically with multiple guardians in the initial guardian set
    // Currently the core contract is setup with only 1 guardian in the set
    const parsedVaa = parse(vaa);
    parsedVaa.signatures = [parsedVaa.signatures[0]];
    vaa = Buffer.from(serialiseVAA(parsedVaa as VAA<Payload>), "hex");
  }

  // Get package IDs
  const coreBridgePackageId = await getPackageId(
    client,
    coreBridgeStateObjectId
  );
  const tokenBridgePackageId = await getPackageId(
    client,
    tokenBridgeStateObjectId
  );

  // Register chain
  let tx = transaction;
  if (!tx) {
    tx = new Transaction();
    tx.setGasBudget(1000000);
  }

  // Get VAA
  const [verifiedVaa] = tx.moveCall({
    target: `${coreBridgePackageId}::vaa::parse_and_verify`,
    arguments: [
      tx.object(coreBridgeStateObjectId),
      tx.pure("vector<u8>", [...vaa]),
      tx.object(SUI_CLOCK_OBJECT_ID),
    ],
  });

  // Get decree ticket
  const [decreeTicket] = tx.moveCall({
    target: `${tokenBridgePackageId}::register_chain::authorize_governance`,
    arguments: [tx.object(tokenBridgeStateObjectId)],
  });

  // Get decree receipt
  const [decreeReceipt] = tx.moveCall({
    target: `${coreBridgePackageId}::governance_message::verify_vaa`,
    arguments: [tx.object(coreBridgeStateObjectId), verifiedVaa, decreeTicket],
    typeArguments: [
      `${tokenBridgePackageId}::register_chain::GovernanceWitness`,
    ],
  });

  // Register chain
  tx.moveCall({
    target: `${tokenBridgePackageId}::register_chain::register_chain`,
    arguments: [tx.object(tokenBridgeStateObjectId), decreeReceipt],
  });

  return tx;
};

/**
 * Currently, (Sui SDK version 0.32.2 and Sui 1.0.0 testnet), there is a
 * mismatch in the max gas budget that causes an error when executing a
 * transaction. Because these values are hardcoded, we set the max gas budget
 * as a temporary workaround.
 * @param network
 * @param tx
 */
export const setMaxGasBudgetDevnet = (network: Network, tx: Transaction) => {
  if (network === "Devnet" || network === "Testnet") {
    // Avoid Error checking transaction input objects: GasBudgetTooHigh { gas_budget: 50000000000, max_budget: 10000000000 }
    tx.setGasBudget(10000000000);
  }
};
