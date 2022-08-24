import { Metric } from "prom-client";

export type CommonEnvironment = {
  logLevel: string;
  promPort: number;
  readinessPort?: number;
  logDir?: string;
  redisHost: string;
  redisPort: number;
  plugins: [
    {
      uri: string;
      overrides: { [key: string]: any };
    }
  ];
  envType: EnvTypes | string;
};

export enum EnvTypes {
  MAIN_NET,
  DEV_NET,
}

export interface WorkerAction {
  chainId: ChainId;
  id: ActionId;
  data: Object;
  description?: string;
  depedencies?: ActionId[];
}

export type ActionId = "UUID"; // todo: import real type

export type ContractFilter = {
  emitterAddress: string;
  chainId: ChainId;
};

export type EVMToolbox = {};

export type SolanaToolbox = {};

export type CosmToolbox = {};

//TODO add loggers

//TODO scheduler w/ staging area for when multiple VAAs are rolling in
export interface Executor {
  relayEvmAction?: (
    walletToolbox: EVMToolbox,
    action: WorkerAction,
    queuedActions: WorkerAction
  ) => ActionQueueUpdate;
  relaySolanaAction?: (
    walletToolbox: SolanaToolbox,
    action: WorkerAction,
    queuedActions: WorkerAction
  ) => ActionQueueUpdate;
  relayCosmAction?: (
    walletToolbox: CosmToolbox,
    action: WorkerAction,
    queuedActions: WorkerAction
  ) => ActionQueueUpdate;
}

export type ListenerEnv = {};

export type ExecutorEnv = {};

export type Plugin = Listener & Executor;
export interface PluginFactory {
  create(config: CommonEnvironment, overrides?: any): Plugin;
}

export abstract class DefaultPlugin implements Plugin {
  defaultConfigs: Map<EnvTypes | string, any> = new Map([]);
  environment: any;
  name: string;

  constructor(config: CommonEnvironment, overrides?: any) {
    const env = {
      ...this.defaultConfigs.get(config.envType),
      ...overrides,
    } as any | undefined;

    if (!env) {
      throw new Error("Plugin config must be defined");
    }
    this.environment = env;
  }

  abstract defineMetrics<T extends string>(): Metric<T>[];

  abstract getFilters(): ContractFilter[];
  abstract consumeEvent(
    vaa: Uint8Array,
    stagingArea: Uint8Array[]
  ): ActionQueueUpdate[];
  relayEvmAction?:
    | ((
        walletToolbox: EVMToolbox,
        action: WorkerAction,
        queuedActions: WorkerAction
      ) => ActionQueueUpdate)
    | undefined;
  relaySolanaAction?:
    | ((
        walletToolbox: SolanaToolbox,
        action: WorkerAction,
        queuedActions: WorkerAction
      ) => ActionQueueUpdate)
    | undefined;
  relayCosmAction?:
    | ((
        walletToolbox: CosmToolbox,
        action: WorkerAction,
        queuedActions: WorkerAction
      ) => ActionQueueUpdate)
    | undefined;
}

export abstract class Listener {
  abstract getFilters(): ContractFilter[];
  abstract consumeEvent(
    vaa: Uint8Array,
    stagingArea: Uint8Array[]
  ): ActionQueueUpdate[];
}

export type ActionQueueUpdate = {
  enqueueActions: WorkerAction[];
  removeActionIds: string[];
};

// todo: import from sdk
export type ChainId = number;
