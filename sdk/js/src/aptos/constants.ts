export enum AptosNetwork {
  Mainnet = "MAINNET",
  Testnet = "TESTNET",
  Devnet = "DEVNET",
}

export const CONTRACT_ADDRESSES = {
  [AptosNetwork.Mainnet]: {
    token_bridge: "",
    core: "",
  },
  [AptosNetwork.Testnet]: {
    token_bridge: "0xb4ec6ea1bff962721cc376be9a9aea840c485f9ceb10f31f523828fd6f4ca95a",
    core: "0x25de93a587d5dc2d7e673663554b7e1d5b00de5d1d38341a896a2141bba5c5c9",
  },
  [AptosNetwork.Devnet]: {
    token_bridge: "0x4450040bc7ea55def9182559ceffc0652d88541538b30a43477364f475f4a4ed",
    core: "0x251011524cd0f76881f16e7c2d822f0c1c9510bfd2430ba24e1b3d52796df204",
  },
};
