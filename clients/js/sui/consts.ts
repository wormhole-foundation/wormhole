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
      "0x1d8a273e7c7de53925aed3fc770466f40fbf2a1ce86dde4cf5434ba9084d45dd",
  },
};

export type SuiAddresses = typeof SUI_OBJECT_IDS;
