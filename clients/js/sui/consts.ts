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
      "0x3f1cf66ea19cbef95205a36c69b6bc04a6097174f47a84944072b63983a0a63c",
    token_bridge_state:
      "0x92c5755e78b9462a9f0cef556dc8e5b3d80d81bc7738bd8104bcabcf50a4a3c6",
  },
};

export type SuiAddresses = typeof SUI_OBJECT_IDS;
