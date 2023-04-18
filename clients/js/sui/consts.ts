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
      "0xc9a5d6d04aec996e60efe6d675b3c9c8ecddd4544f20474916771b4f90b4b0dc",
  },
};

export type SuiAddresses = typeof SUI_OBJECT_IDS;
