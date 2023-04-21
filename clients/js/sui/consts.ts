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
      "0x50d49cf0c8f0ab33b0c4ad1693a2617f6b4fe4dac3e6e2d0ce6e9fbe83795b51",
    token_bridge_state:
      "0x546ee2833042967392ceeeca5a94a9070adad5331dcfa3584749f4aa8a285fe7",
  },
};

export type SuiAddresses = typeof SUI_OBJECT_IDS;
