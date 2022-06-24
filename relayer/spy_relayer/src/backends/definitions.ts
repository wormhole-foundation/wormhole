import { ChainId } from "@certusone/wormhole-sdk";
import { ScopedLogger } from "../helpers/logHelper";
import { PromHelper } from "../helpers/promHelpers";
import { StoreKey, StorePayload } from "../helpers/redisHelper";

/** TypedFilter is used by subscribeSignedVAA to filter messages returned by the guardian spy */
export interface TypedFilter {
  emitterFilter: { chainId: ChainId; emitterAddress: string };
}

/** Listen to VAAs via a http listener or guardian spy service */
export interface Listener {
  logger: ScopedLogger;

  /** Get filters for the guardian spy subscription */
  getEmitterFilters(): Promise<TypedFilter[]>;

  /** Parse and validate the received VAAs from the spy */
  validate(rawVaa: Uint8Array): Promise<unknown>;

  /** Process and add the VAA to redis if it is valid */
  process(rawVaa: Uint8Array): Promise<void>;

  /** Serialize and store a validated VAA in redis for the relayer */
  store(key: StoreKey, payload: StorePayload): Promise<void>;
}

/** Relayer is an interface for relaying messages across chains */
export interface Relayer {
  /** For the audit thread to confirm the relay is completed and no chain rollbacks or reorgs have occurred */
  isComplete(): boolean; // For the audit thread

  /** Parse the payload and return the target chain id */
  targetChainId(): ChainId;

  /** Process the request to relay a message */
  process(key: string, privKey: any, logger: ScopedLogger): Promise<void>;
}

/** Backend is the interface necessary to implement for custom relayers */
export interface Backend {
  listener: Listener;
  relayer: Relayer;
}
