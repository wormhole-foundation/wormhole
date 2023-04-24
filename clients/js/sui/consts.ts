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
      "0xb8d8fd9b805e4d1aea5c1edcc2e4097b99f3335ff673cfe01d2dbaa54aa564c2",
    token_bridge_state:
      "0xcea50dafc22685e39b1469e12e4326788279d8e4108bfd2e2a59897199c65f50",
  },
};

export type SuiAddresses = typeof SUI_OBJECT_IDS;
