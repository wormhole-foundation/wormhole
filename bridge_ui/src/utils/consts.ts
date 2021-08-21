import {
  ChainId,
  CHAIN_ID_BSC,
  CHAIN_ID_ETH,
  CHAIN_ID_SOLANA,
  CHAIN_ID_TERRA,
} from "@certusone/wormhole-sdk";
import { clusterApiUrl } from "@solana/web3.js";
import { getAddress } from "ethers/lib/utils";

export interface ChainInfo {
  id: ChainId;
  name: string;
}
export const CHAINS =
  process.env.REACT_APP_CLUSTER === "testnet"
    ? [
        {
          id: CHAIN_ID_ETH,
          name: "Ethereum",
        },
        {
          id: CHAIN_ID_SOLANA,
          name: "Solana",
        },
      ]
    : [
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
export const WORMHOLE_RPC_HOST =
  process.env.REACT_APP_CLUSTER === "testnet"
    ? "https://wormhole-v2-testnet-api.certus.one"
    : "http://localhost:8080";
export const SOLANA_HOST =
  process.env.REACT_APP_CLUSTER === "testnet"
    ? clusterApiUrl("testnet")
    : "http://localhost:8899";
export const TERRA_HOST = "http://localhost:1317";
export const ETH_TEST_TOKEN_ADDRESS = getAddress(
  process.env.REACT_APP_CLUSTER === "testnet"
    ? "0xcEE940033DA197F551BBEdED7F4aA55Ee55C582B"
    : "0x67B5656d60a809915323Bf2C40A8bEF15A152e3e"
);
export const ETH_BRIDGE_ADDRESS = getAddress(
  process.env.REACT_APP_CLUSTER === "testnet"
    ? "0x44F3e7c20850B3B5f3031114726A9240911D912a"
    : "0xC89Ce4735882C9F0f0FE26686c53074E09B0D550"
);
export const ETH_TOKEN_BRIDGE_ADDRESS = getAddress(
  process.env.REACT_APP_CLUSTER === "testnet"
    ? "0xa6CDAddA6e4B6704705b065E01E52e2486c0FBf6"
    : "0x0290FB167208Af455bB137780163b7B7a9a10C16"
);
export const SOL_TEST_TOKEN_ADDRESS =
  process.env.REACT_APP_CLUSTER === "testnet"
    ? "6uzMjLkcTwhYo5Fwx9DtVtQ7VRrCQ7bTUd7rHXTiPDXp"
    : "2WDq7wSs9zYrpx2kbHDA4RUTRch2CCTP6ZWaH4GNfnQQ";
export const SOL_BRIDGE_ADDRESS =
  process.env.REACT_APP_CLUSTER === "testnet"
    ? "H3SjyXYezgWj1ktCrazLkD8ydj9gzmEi4w9zRCXg2G4R"
    : "Bridge1p5gheXUvJ6jGWGeCsgPKgnE3YgdGKRVCMY9o";
export const SOL_TOKEN_BRIDGE_ADDRESS =
  process.env.REACT_APP_CLUSTER === "testnet"
    ? "ToknwtcmUawaJk2pxSwZzJRrgReH52a3QRDE1Mgid9b"
    : "B6RHG3mfcckmrYN1UhmJzyS1XX3fZKbkeUcpJe9Sy3FE";
export const TERRA_TEST_TOKEN_ADDRESS =
  "terra13nkgqrfymug724h8pprpexqj9h629sa3ncw7sh";
export const TERRA_BRIDGE_ADDRESS =
  "terra18vd8fpwxzck93qlwghaj6arh4p7c5n896xzem5";
export const TERRA_TOKEN_BRIDGE_ADDRESS =
  "terra10pyejy66429refv3g35g2t7am0was7ya7kz2a4";
