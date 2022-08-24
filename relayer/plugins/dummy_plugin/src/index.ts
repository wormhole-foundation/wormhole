import {
  ActionQueueUpdate,
  CommonEnvironment,
  ContractFilter,
  CosmToolbox,
  EVMToolbox,
  Plugin,
  PluginFactory,
  SolanaToolbox,
  WorkerAction,
} from "plugin_interface";

export function create(config: CommonEnvironment, overrides?: any): Plugin {
  console.log("Creating da plugin...");
  return new DummyPlugin(config, overrides);
}

class DummyPlugin implements Plugin {
  constructor(config: CommonEnvironment, overrides: Object) {
    console.log(`Config: ${JSON.stringify(config, undefined, 2)}`);
    console.log(`Overrides: ${JSON.stringify(overrides, undefined, 2)}`);
  }

  getFilters(): ContractFilter[] {
    return [{ chainId: 1, emitterAddress: "gotcha!!" }];
  }
  consumeEvent(
    vaa: Uint8Array,
    stagingArea: Uint8Array[]
  ): ActionQueueUpdate[] {
    return [];
  }
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

export default { create };
