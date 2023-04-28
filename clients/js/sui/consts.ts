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
      "0xc1867890b51a1fe873ce34fc4ebc6d87e1ebe30b340d9adccf77e38cf8f2453b",
    token_bridge_state:
      "0xfbbe04379273ad9e09463c47c678542a8bf95a3c8e765f42d1adf5a8f54ea9bc",
  },
};

export type SuiAddresses = typeof SUI_OBJECT_IDS;
