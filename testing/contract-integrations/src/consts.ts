import {
  ChainId,
  CHAIN_ID_BSC,
  CHAIN_ID_ETH,
  CHAIN_ID_POLYGON,
  CHAIN_ID_SOLANA,
  CHAIN_ID_TERRA,
} from "@certusone/wormhole-sdk";
import { Signer } from "@ethersproject/abstract-signer";
import { clusterApiUrl } from "@solana/web3.js";
import { ethers } from "ethers";
import { getAddress } from "ethers/lib/utils";

//Devnet here means the localhost kubernetes environment used by the wormhole-foundation/wormhole official git repository.
//Testnet is the official Wormhole testnet
export type Environment = "devnet" | "testnet" | "mainnet";
export const CLUSTER: Environment = "devnet" as Environment; //This is the currently selected environment.

export const SOLANA_HOST = process.env.REACT_APP_SOLANA_API_URL
  ? process.env.REACT_APP_SOLANA_API_URL
  : CLUSTER === "mainnet"
  ? clusterApiUrl("mainnet-beta")
  : CLUSTER === "testnet"
  ? clusterApiUrl("testnet")
  : "http://localhost:8899";

export const TERRA_HOST =
  CLUSTER === "mainnet"
    ? {
        URL: "https://lcd.terra.dev",
        chainID: "columbus-5",
        name: "mainnet",
      }
    : CLUSTER === "testnet"
    ? {
        URL: "https://bombay-lcd.terra.dev",
        chainID: "bombay-12",
        name: "testnet",
      }
    : {
        URL: "http://localhost:1317",
        chainID: "columbus-5",
        name: "localterra",
      };
export const ETH_BRIDGE_ADDRESS = getAddress(
  CLUSTER === "mainnet"
    ? "0x98f3c9e6E3fAce36bAAd05FE09d375Ef1464288B"
    : CLUSTER === "testnet"
    ? "0x44F3e7c20850B3B5f3031114726A9240911D912a"
    : "0xC89Ce4735882C9F0f0FE26686c53074E09B0D550"
);
export const ETH_NFT_BRIDGE_ADDRESS = getAddress(
  CLUSTER === "mainnet"
    ? "0x6FFd7EdE62328b3Af38FCD61461Bbfc52F5651fE"
    : CLUSTER === "testnet"
    ? "0x26b4afb60d6c903165150c6f0aa14f8016be4aec" // TODO: test address
    : "0x26b4afb60d6c903165150c6f0aa14f8016be4aec"
);
export const ETH_TOKEN_BRIDGE_ADDRESS = getAddress(
  CLUSTER === "mainnet"
    ? "0x3ee18B2214AFF97000D974cf647E7C347E8fa585"
    : CLUSTER === "testnet"
    ? "0xa6CDAddA6e4B6704705b065E01E52e2486c0FBf6"
    : "0x0290FB167208Af455bB137780163b7B7a9a10C16"
);
export const BSC_BRIDGE_ADDRESS = getAddress(
  CLUSTER === "mainnet"
    ? "0x98f3c9e6E3fAce36bAAd05FE09d375Ef1464288B"
    : CLUSTER === "testnet"
    ? "0xC89Ce4735882C9F0f0FE26686c53074E09B0D550" // TODO: test address
    : "0xC89Ce4735882C9F0f0FE26686c53074E09B0D550"
);
export const BSC_NFT_BRIDGE_ADDRESS = getAddress(
  CLUSTER === "mainnet"
    ? "0x5a58505a96D1dbf8dF91cB21B54419FC36e93fdE"
    : CLUSTER === "testnet"
    ? "0x26b4afb60d6c903165150c6f0aa14f8016be4aec" // TODO: test address
    : "0x26b4afb60d6c903165150c6f0aa14f8016be4aec"
);
export const BSC_TOKEN_BRIDGE_ADDRESS = getAddress(
  CLUSTER === "mainnet"
    ? "0xB6F6D86a8f9879A9c87f643768d9efc38c1Da6E7"
    : CLUSTER === "testnet"
    ? "0x0290FB167208Af455bB137780163b7B7a9a10C16" // TODO: test address
    : "0x0290FB167208Af455bB137780163b7B7a9a10C16"
);
export const POLYGON_BRIDGE_ADDRESS = getAddress(
  CLUSTER === "mainnet"
    ? "0x7A4B5a56256163F07b2C80A7cA55aBE66c4ec4d7"
    : CLUSTER === "testnet"
    ? "0xC89Ce4735882C9F0f0FE26686c53074E09B0D550" // TODO: test address
    : "0xC89Ce4735882C9F0f0FE26686c53074E09B0D550"
);
export const POLYGON_NFT_BRIDGE_ADDRESS = getAddress(
  CLUSTER === "mainnet"
    ? "0x90BBd86a6Fe93D3bc3ed6335935447E75fAb7fCf"
    : CLUSTER === "testnet"
    ? "0x26b4afb60d6c903165150c6f0aa14f8016be4aec" // TODO: test address
    : "0x26b4afb60d6c903165150c6f0aa14f8016be4aec"
);
export const POLYGON_TOKEN_BRIDGE_ADDRESS = getAddress(
  CLUSTER === "mainnet"
    ? "0x5a58505a96D1dbf8dF91cB21B54419FC36e93fdE"
    : CLUSTER === "testnet"
    ? "0x0290FB167208Af455bB137780163b7B7a9a10C16" // TODO: test address
    : "0x0290FB167208Af455bB137780163b7B7a9a10C16"
);
export const SOL_BRIDGE_ADDRESS =
  CLUSTER === "mainnet"
    ? "worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth"
    : CLUSTER === "testnet"
    ? "Brdguy7BmNB4qwEbcqqMbyV5CyJd2sxQNUn6NEpMSsUb"
    : "Bridge1p5gheXUvJ6jGWGeCsgPKgnE3YgdGKRVCMY9o";
export const SOL_NFT_BRIDGE_ADDRESS =
  CLUSTER === "mainnet"
    ? "WnFt12ZrnzZrFZkt2xsNsaNWoQribnuQ5B5FrDbwDhD"
    : CLUSTER === "testnet"
    ? "NFTWqJR8YnRVqPDvTJrYuLrQDitTG5AScqbeghi4zSA" // TODO: test address
    : "NFTWqJR8YnRVqPDvTJrYuLrQDitTG5AScqbeghi4zSA";
export const SOL_TOKEN_BRIDGE_ADDRESS =
  CLUSTER === "mainnet"
    ? "wormDTUJ6AWPNvk59vGQbDvGJmqbDTdgWgAqcLBCgUb"
    : CLUSTER === "testnet"
    ? "A4Us8EhCC76XdGAN17L4KpRNEK423nMivVHZzZqFqqBg"
    : "B6RHG3mfcckmrYN1UhmJzyS1XX3fZKbkeUcpJe9Sy3FE";
export const TERRA_BRIDGE_ADDRESS =
  CLUSTER === "mainnet"
    ? "terra1dq03ugtd40zu9hcgdzrsq6z2z4hwhc9tqk2uy5"
    : CLUSTER === "testnet"
    ? "terra14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9ssrc8au"
    : "terra14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9ssrc8au";
export const TERRA_TOKEN_BRIDGE_ADDRESS =
  CLUSTER === "mainnet"
    ? "terra10nmmwe8r3g99a9newtqa7a75xfgs2e8z87r2sf"
    : CLUSTER === "testnet"
    ? "terra1nc5tatafv6eyq7llkr2gv50ff9e22mnf70qgjlv737ktmt4eswrquka9l6"
    : "terra1nc5tatafv6eyq7llkr2gv50ff9e22mnf70qgjlv737ktmt4eswrquka9l6";

export const getBridgeAddressForChain = (chainId: ChainId) =>
  chainId === CHAIN_ID_SOLANA
    ? SOL_BRIDGE_ADDRESS
    : chainId === CHAIN_ID_ETH
    ? ETH_BRIDGE_ADDRESS
    : chainId === CHAIN_ID_BSC
    ? BSC_BRIDGE_ADDRESS
    : chainId === CHAIN_ID_TERRA
    ? TERRA_BRIDGE_ADDRESS
    : chainId === CHAIN_ID_POLYGON
    ? POLYGON_BRIDGE_ADDRESS
    : "";
export const getNFTBridgeAddressForChain = (chainId: ChainId) =>
  chainId === CHAIN_ID_SOLANA
    ? SOL_NFT_BRIDGE_ADDRESS
    : chainId === CHAIN_ID_ETH
    ? ETH_NFT_BRIDGE_ADDRESS
    : chainId === CHAIN_ID_BSC
    ? BSC_NFT_BRIDGE_ADDRESS
    : chainId === CHAIN_ID_POLYGON
    ? POLYGON_NFT_BRIDGE_ADDRESS
    : "";
export const getTokenBridgeAddressForChain = (chainId: ChainId) =>
  chainId === CHAIN_ID_SOLANA
    ? SOL_TOKEN_BRIDGE_ADDRESS
    : chainId === CHAIN_ID_ETH
    ? ETH_TOKEN_BRIDGE_ADDRESS
    : chainId === CHAIN_ID_BSC
    ? BSC_TOKEN_BRIDGE_ADDRESS
    : chainId === CHAIN_ID_TERRA
    ? TERRA_TOKEN_BRIDGE_ADDRESS
    : chainId === CHAIN_ID_POLYGON
    ? POLYGON_TOKEN_BRIDGE_ADDRESS
    : "";

export const WORMHOLE_RPC_HOSTS =
  CLUSTER === "mainnet"
    ? [
        "https://wormhole-v2-mainnet-api.certus.one",
        "https://wormhole.inotel.ro",
        "https://wormhole-v2-mainnet-api.mcf.rocks",
        "https://wormhole-v2-mainnet-api.chainlayer.network",
        "https://wormhole-v2-mainnet-api.staking.fund",
        "https://wormhole-v2-mainnet.01node.com",
      ]
    : CLUSTER === "testnet"
    ? ["https://wormhole-v2-testnet-api.certus.one"]
    : ["http://localhost:7071"];

export const ETH_NODE_URL = "ws://localhost:8545"; //TODO testnet
export const POLYGON_NODE_URL = "ws:localhost:0000"; //TODO
export const BSC_NODE_URL = "ws://localhost:8546"; //TODO testnet
export const ETH_PRIVATE_KEY =
  "0x4f3edf983ac636a65a842ce7c78d9aa706d3b113bce9c46f30d7d21715b23b1d";

export const SOLANA_PRIVATE_KEY = new Uint8Array([
  14, 173, 153, 4, 176, 224, 201, 111, 32, 237, 183, 185, 159, 247, 22, 161, 89,
  84, 215, 209, 212, 137, 10, 92, 157, 49, 29, 192, 101, 164, 152, 70, 87, 65,
  8, 174, 214, 157, 175, 126, 98, 90, 54, 24, 100, 177, 247, 77, 19, 112, 47,
  44, 165, 109, 233, 102, 14, 86, 109, 29, 134, 145, 132, 141,
]);

export function getSignerForChain(chainId: ChainId): Signer {
  const provider = new ethers.providers.WebSocketProvider(
    chainId === CHAIN_ID_POLYGON
      ? POLYGON_NODE_URL
      : chainId === CHAIN_ID_BSC
      ? BSC_NODE_URL
      : ETH_NODE_URL
  );
  const signer = new ethers.Wallet(ETH_PRIVATE_KEY, provider as any);
  return signer;
}

export const ETH_TEST_WALLET_PUBLIC_KEY =
  "0x90F8bf6A479f320ead074411a4B0e7944Ea8c9C1";

export const ETH_TEST_TOKEN = "0x2D8BE6BF0baA74e0A907016679CaE9190e80dD0A";

export const SOLANA_TEST_TOKEN = "2WDq7wSs9zYrpx2kbHDA4RUTRch2CCTP6ZWaH4GNfnQQ"; //SOLT on devnet
export const SOLANA_TEST_WALLET_PUBLIC_KEY =
  "6sbzC1eH4FTujJXWj51eQe25cYvr4xfXbJ1vAj7j2k5J";
