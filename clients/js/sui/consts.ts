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
      "0x6128b6adb677ac2da9ac5efb3003e5863825748f7a18786c5f612a4cb552fa50",
    token_bridge_state:
      "0xff8d34100d23d54c48c662aa0def908b42048e1d84d1e9034fcb1cb91f5704aa",
  },
};

export type SuiAddresses = typeof SUI_OBJECT_IDS;
