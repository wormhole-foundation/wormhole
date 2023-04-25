/**
 * On Sui, we must hardcode both the package ID as well as the object IDs of
 * the State objects created when we initialize the core and token bridges.
 *
 * TODO(aki): move this to SDK at some point
 */
export const SUI_OBJECT_IDS = {
  MAINNET: {
    core_state: undefined,
    token_bridge_state: undefined,
  },
  TESTNET: {
    core_state: undefined,
    token_bridge_state: undefined,
  },
  DEVNET: {
    core_state:
      "0x59e9ce8f55cbd4f6b8fade7568e8ff7e594c1a37ec78eb126ebebe20d0722df2",
    token_bridge_state:
      "0x7ffaddf3eda80ceafa0d958ff61a4f3532f165d6efa9ae4af2d3914db76c6a9d",
  },
};

export type SuiAddresses = typeof SUI_OBJECT_IDS;
