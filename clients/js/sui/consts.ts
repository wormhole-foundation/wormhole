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
      "0xa136cdd2fbf6eeac52bcddf882c55daf8b0205b549a7b106a4704afcf9c85e6f",
    token_bridge_state:
      "0xf0d80ed63eb4758e8b38af9519af41e82823d703a81d323970163fcf44f552c8",
  },
};

export type SuiAddresses = typeof SUI_OBJECT_IDS;
