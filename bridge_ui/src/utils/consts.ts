import {
  ChainId,
  CHAIN_ID_BSC,
  CHAIN_ID_ETH,
  CHAIN_ID_SOLANA,
  CHAIN_ID_TERRA,
} from "@certusone/wormhole-sdk";
import { getAddress } from "ethers/lib/utils";

export interface ChainInfo {
  id: ChainId;
  name: string;
}
export const CHAINS = [
  {
    id: CHAIN_ID_BSC,
    name: "Binance Smart Chain",
  },
  {
    id: CHAIN_ID_ETH,
    name: "Ethereum",
  },
  {
    id: CHAIN_ID_SOLANA,
    name: "Solana",
  },
  {
    id: CHAIN_ID_TERRA,
    name: "Terra",
  },
];
export type ChainsById = { [key in ChainId]: ChainInfo };
export const CHAINS_BY_ID: ChainsById = CHAINS.reduce((obj, chain) => {
  obj[chain.id] = chain;
  return obj;
}, {} as ChainsById);
export const WORMHOLE_RPC_HOST = "http://localhost:8080";
export const SOLANA_HOST = "http://localhost:8899";
export const ETH_TEST_TOKEN_ADDRESS = getAddress(
  "0x0290FB167208Af455bB137780163b7B7a9a10C16"
);
export const ETH_BRIDGE_ADDRESS = getAddress(
  "0x254dffcd3277c0b1660f6d42efbb754edababc2b"
);
export const ETH_TOKEN_BRIDGE_ADDRESS = getAddress(
  "0xe982e462b094850f12af94d21d470e21be9d0e9c"
);
export const SOL_TEST_TOKEN_ADDRESS =
  "2WDq7wSs9zYrpx2kbHDA4RUTRch2CCTP6ZWaH4GNfnQQ";
export const SOL_BRIDGE_ADDRESS = "Bridge1p5gheXUvJ6jGWGeCsgPKgnE3YgdGKRVCMY9o";
export const SOL_TOKEN_BRIDGE_ADDRESS =
  "B6RHG3mfcckmrYN1UhmJzyS1XX3fZKbkeUcpJe9Sy3FE";
