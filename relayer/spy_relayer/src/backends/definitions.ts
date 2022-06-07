import { ChainId } from "@certusone/wormhole-sdk";
import { getLogger, getScopedLogger, ScopedLogger } from "../helpers/logHelper";
import { ListenerEnvironment } from "../configureEnv";

/** TypedFilter is used by subscribeSignedVAA to filter messages returned by the guardian spy */
export interface TypedFilter {
  emitterFilter: { chainId: ChainId; emitterAddress: string };
}

/** Listener is an interface for listening for VAAs for a given type */
export interface Listener {
  logger: ScopedLogger
  env: ListenerEnvironment
  getEmitterFilters(): Promise<TypedFilter[]>
  shouldRelay(rawVaa: Uint8Array): Promise<boolean>
}

/** Relayer is an interface for relaying messages across chains */
export interface Relayer {
  logger: ScopedLogger

  run(): void
  isComplete(): boolean; // For the audit thread
  targetChain(): ChainId; // Parse payload for target chain: relay_worker.ts:processRequest()
}

/** Backend is the interface necessary to implement for custom relayers */
export interface Backend {
    listener: Listener
    relayer: Relayer
}