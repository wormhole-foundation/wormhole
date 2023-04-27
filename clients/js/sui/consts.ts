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
      "0x2647204bd89dbc9f9489e07cebc1e2e579b70c10963dd758b4c59de59ae966d7",
    token_bridge_state:
      "0xf0c03c4492c61abd73b1cc139589fbd96e3b95c502d6f47cfa59637d320968b0",
  },
};

export type SuiAddresses = typeof SUI_OBJECT_IDS;
