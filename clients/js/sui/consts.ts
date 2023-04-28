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
      "0x9ed35784102112cd126d762a314ae0dea22a2d40524d743e9b2c6ab66475be5d",
    token_bridge_state:
      "0xcdc4bcc4a8eca476cdc57770ab9564f5b88354dfad119222345a4f8ac8146184",
  },
};

export type SuiAddresses = typeof SUI_OBJECT_IDS;
