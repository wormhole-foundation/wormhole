import { ChainId } from "@certusone/wormhole-sdk";




// signer
export const ORACLE_DEPLOYER_PRIVATE_KEY = "0x4f3edf983ac636a65a842ce7c78d9aa706d3b113bce9c46f30d7d21715b23b1d";
export const RELAYER_DEPLOYER_PRIVATE_KEY = "0x4f3edf983ac636a65a842ce7c78d9aa706d3b113bce9c46f30d7d21715b23b1d";
export const GUARDIAN_PRIVATE_KEY = "cfb12303a19cde580bb4dd771639b0d26bc68353645571a8cff516ab2ee113a0";

// wormhole event ABIs
export const WORMHOLE_MESSAGE_EVENT_ABI = [
  "event LogMessagePublished(address indexed sender, uint64 sequence, uint32 nonce, bytes payload, uint8 consistencyLevel)",
];

export type ChainInfo = {
  evmNetworkId: number
  chainId: ChainId
  rpc: string
  wormholeAddress: string
  coreRelayerAddress: string
  description: string
};