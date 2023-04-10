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
      "0x363f879229a8c71bdf2c739864d236d30437acc01b664ffd4799037398861440",
    token_bridge_state:
      "0xd194d83e1e8839ec80179fd7b5f0d14de4a906adda3d8f119c24ee0c8f3964f9",
  },
};

export type SuiAddresses = typeof SUI_OBJECT_IDS;
