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
      "0xd638fbc5edeca2741949c2c0ca07bedf387aa51e8603b637138808089c6cc17e",
    token_bridge_state:
      "0x7ead9c4b06db0e64029698a6e2d83ca2eb7126a912556193dd3b606b1fa3d237",
  },
};

export type SuiAddresses = typeof SUI_OBJECT_IDS;
