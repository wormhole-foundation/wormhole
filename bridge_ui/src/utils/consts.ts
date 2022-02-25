import {
  ChainId,
  CHAIN_ID_AVAX,
  CHAIN_ID_BSC,
  CHAIN_ID_ETH,
  CHAIN_ID_ETHEREUM_ROPSTEN,
  CHAIN_ID_POLYGON,
  CHAIN_ID_SOLANA,
  CHAIN_ID_TERRA,
  CHAIN_ID_OASIS,
  isEVMChain,
} from "@certusone/wormhole-sdk";
import { clusterApiUrl } from "@solana/web3.js";
import { getAddress } from "ethers/lib/utils";
import avaxIcon from "../icons/avax.svg";
import bscIcon from "../icons/bsc.svg";
import ethIcon from "../icons/eth.svg";
import oasisIcon from "../icons/oasis-network-rose-logo.svg";
import polygonIcon from "../icons/polygon.svg";
import solanaIcon from "../icons/solana.svg";
import terraIcon from "../icons/terra.svg";

export type Cluster = "devnet" | "testnet" | "mainnet";
export const CLUSTER: Cluster =
  process.env.REACT_APP_CLUSTER === "mainnet"
    ? "mainnet"
    : process.env.REACT_APP_CLUSTER === "testnet"
    ? "testnet"
    : "devnet";
export interface ChainInfo {
  id: ChainId;
  name: string;
  logo: string;
}
export const CHAINS: ChainInfo[] =
  CLUSTER === "mainnet"
    ? [
        {
          id: CHAIN_ID_AVAX,
          name: "Avalanche",
          logo: avaxIcon,
        },
        {
          id: CHAIN_ID_OASIS,
          name: "Oasis",
          logo: oasisIcon,
        },
        {
          id: CHAIN_ID_BSC,
          name: "Binance Smart Chain",
          logo: bscIcon,
        },
        {
          id: CHAIN_ID_ETH,
          name: "Ethereum",
          logo: ethIcon,
        },
        {
          id: CHAIN_ID_POLYGON,
          name: "Polygon",
          logo: polygonIcon,
        },
        {
          id: CHAIN_ID_SOLANA,
          name: "Solana",
          logo: solanaIcon,
        },
        {
          id: CHAIN_ID_TERRA,
          name: "Terra",
          logo: terraIcon,
        },
      ]
    : CLUSTER === "testnet"
    ? [
        {
          id: CHAIN_ID_AVAX,
          name: "Avalanche",
          logo: avaxIcon,
        },
        {
          id: CHAIN_ID_OASIS,
          name: "Oasis",
          logo: oasisIcon,
        },
        {
          id: CHAIN_ID_BSC,
          name: "Binance Smart Chain",
          logo: bscIcon,
        },
        {
          id: CHAIN_ID_ETH,
          name: "Ethereum (Goerli)",
          logo: ethIcon,
        },
        {
          id: CHAIN_ID_ETHEREUM_ROPSTEN,
          name: "Ethereum (Ropsten)",
          logo: ethIcon,
        },
        {
          id: CHAIN_ID_POLYGON,
          name: "Polygon",
          logo: polygonIcon,
        },
        {
          id: CHAIN_ID_SOLANA,
          name: "Solana",
          logo: solanaIcon,
        },
        {
          id: CHAIN_ID_TERRA,
          name: "Terra",
          logo: terraIcon,
        },
      ]
    : [
        {
          id: CHAIN_ID_BSC,
          name: "Binance Smart Chain",
          logo: bscIcon,
        },
        {
          id: CHAIN_ID_ETH,
          name: "Ethereum",
          logo: ethIcon,
        },
        {
          id: CHAIN_ID_SOLANA,
          name: "Solana",
          logo: solanaIcon,
        },
        {
          id: CHAIN_ID_TERRA,
          name: "Terra",
          logo: terraIcon,
        },
      ];
export const BETA_CHAINS: ChainId[] = CLUSTER === "mainnet" ? [] : [];
export const CHAINS_WITH_NFT_SUPPORT = CHAINS.filter(
  ({ id }) =>
    id === CHAIN_ID_AVAX ||
    id === CHAIN_ID_BSC ||
    id === CHAIN_ID_ETH ||
    id === CHAIN_ID_ETHEREUM_ROPSTEN ||
    id === CHAIN_ID_POLYGON ||
    id === CHAIN_ID_OASIS ||
    id === CHAIN_ID_SOLANA
);
export type ChainsById = { [key in ChainId]: ChainInfo };
export const CHAINS_BY_ID: ChainsById = CHAINS.reduce((obj, chain) => {
  obj[chain.id] = chain;
  return obj;
}, {} as ChainsById);

export const COMING_SOON_CHAINS: ChainInfo[] = [];
export const getDefaultNativeCurrencySymbol = (chainId: ChainId) =>
  chainId === CHAIN_ID_SOLANA
    ? "SOL"
    : chainId === CHAIN_ID_ETH || chainId === CHAIN_ID_ETHEREUM_ROPSTEN
    ? "ETH"
    : chainId === CHAIN_ID_BSC
    ? "BNB"
    : chainId === CHAIN_ID_TERRA
    ? "LUNA"
    : chainId === CHAIN_ID_POLYGON
    ? "MATIC"
    : chainId === CHAIN_ID_AVAX
    ? "AVAX"
    : chainId === CHAIN_ID_OASIS
    ? "ROSE"
    : "";
export const getExplorerName = (chainId: ChainId) =>
  chainId === CHAIN_ID_ETH || chainId === CHAIN_ID_ETHEREUM_ROPSTEN
    ? "Etherscan"
    : chainId === CHAIN_ID_BSC
    ? "BscScan"
    : chainId === CHAIN_ID_TERRA
    ? "Finder"
    : chainId === CHAIN_ID_POLYGON
    ? "Polygonscan"
    : chainId === CHAIN_ID_AVAX
    ? "Snowtrace"
    : "Explorer";
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
export const ETH_NETWORK_CHAIN_ID =
  CLUSTER === "mainnet" ? 1 : CLUSTER === "testnet" ? 5 : 1337;
export const ROPSTEN_ETH_NETWORK_CHAIN_ID =
  CLUSTER === "mainnet" ? 1 : CLUSTER === "testnet" ? 3 : 1337;
export const BSC_NETWORK_CHAIN_ID =
  CLUSTER === "mainnet" ? 56 : CLUSTER === "testnet" ? 97 : 1397;
export const POLYGON_NETWORK_CHAIN_ID =
  CLUSTER === "mainnet" ? 137 : CLUSTER === "testnet" ? 80001 : 1381;
export const AVAX_NETWORK_CHAIN_ID =
  CLUSTER === "mainnet" ? 43114 : CLUSTER === "testnet" ? 43113 : 1381;
export const OASIS_NETWORK_CHAIN_ID =
  CLUSTER === "mainnet" ? 42262 : CLUSTER === "testnet" ? 42261 : 1381;
export const getEvmChainId = (chainId: ChainId) =>
  chainId === CHAIN_ID_ETH
    ? ETH_NETWORK_CHAIN_ID
    : chainId === CHAIN_ID_ETHEREUM_ROPSTEN
    ? ROPSTEN_ETH_NETWORK_CHAIN_ID
    : chainId === CHAIN_ID_BSC
    ? BSC_NETWORK_CHAIN_ID
    : chainId === CHAIN_ID_POLYGON
    ? POLYGON_NETWORK_CHAIN_ID
    : chainId === CHAIN_ID_AVAX
    ? AVAX_NETWORK_CHAIN_ID
    : chainId === CHAIN_ID_OASIS
    ? OASIS_NETWORK_CHAIN_ID
    : undefined;
export const SOLANA_HOST = process.env.REACT_APP_SOLANA_API_URL
  ? process.env.REACT_APP_SOLANA_API_URL
  : CLUSTER === "mainnet"
  ? clusterApiUrl("mainnet-beta")
  : CLUSTER === "testnet"
  ? clusterApiUrl("devnet")
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
    ? "0x706abc4E45D419950511e474C7B9Ed348A4a716c"
    : "0xC89Ce4735882C9F0f0FE26686c53074E09B0D550"
);
export const ETH_NFT_BRIDGE_ADDRESS = getAddress(
  CLUSTER === "mainnet"
    ? "0x6FFd7EdE62328b3Af38FCD61461Bbfc52F5651fE"
    : CLUSTER === "testnet"
    ? "0xD8E4C2DbDd2e2bd8F1336EA691dBFF6952B1a6eB"
    : "0x26b4afb60d6c903165150c6f0aa14f8016be4aec"
);
export const ETH_TOKEN_BRIDGE_ADDRESS = getAddress(
  CLUSTER === "mainnet"
    ? "0x3ee18B2214AFF97000D974cf647E7C347E8fa585"
    : CLUSTER === "testnet"
    ? "0xF890982f9310df57d00f659cf4fd87e65adEd8d7"
    : "0x0290FB167208Af455bB137780163b7B7a9a10C16"
);
export const BSC_BRIDGE_ADDRESS = getAddress(
  CLUSTER === "mainnet"
    ? "0x98f3c9e6E3fAce36bAAd05FE09d375Ef1464288B"
    : CLUSTER === "testnet"
    ? "0x68605AD7b15c732a30b1BbC62BE8F2A509D74b4D"
    : "0xC89Ce4735882C9F0f0FE26686c53074E09B0D550"
);
export const BSC_NFT_BRIDGE_ADDRESS = getAddress(
  CLUSTER === "mainnet"
    ? "0x5a58505a96D1dbf8dF91cB21B54419FC36e93fdE"
    : CLUSTER === "testnet"
    ? "0xcD16E5613EF35599dc82B24Cb45B5A93D779f1EE"
    : "0x26b4afb60d6c903165150c6f0aa14f8016be4aec"
);
export const BSC_TOKEN_BRIDGE_ADDRESS = getAddress(
  CLUSTER === "mainnet"
    ? "0xB6F6D86a8f9879A9c87f643768d9efc38c1Da6E7"
    : CLUSTER === "testnet"
    ? "0x9dcF9D205C9De35334D646BeE44b2D2859712A09"
    : "0x0290FB167208Af455bB137780163b7B7a9a10C16"
);
export const POLYGON_BRIDGE_ADDRESS = getAddress(
  CLUSTER === "mainnet"
    ? "0x7A4B5a56256163F07b2C80A7cA55aBE66c4ec4d7"
    : CLUSTER === "testnet"
    ? "0x0CBE91CF822c73C2315FB05100C2F714765d5c20"
    : "0xC89Ce4735882C9F0f0FE26686c53074E09B0D550"
);
export const POLYGON_NFT_BRIDGE_ADDRESS = getAddress(
  CLUSTER === "mainnet"
    ? "0x90BBd86a6Fe93D3bc3ed6335935447E75fAb7fCf"
    : CLUSTER === "testnet"
    ? "0x51a02d0dcb5e52F5b92bdAA38FA013C91c7309A9"
    : "0x26b4afb60d6c903165150c6f0aa14f8016be4aec"
);
export const POLYGON_TOKEN_BRIDGE_ADDRESS = getAddress(
  CLUSTER === "mainnet"
    ? "0x5a58505a96D1dbf8dF91cB21B54419FC36e93fdE"
    : CLUSTER === "testnet"
    ? "0x377D55a7928c046E18eEbb61977e714d2a76472a"
    : "0x0290FB167208Af455bB137780163b7B7a9a10C16"
);
export const AVAX_BRIDGE_ADDRESS = getAddress(
  CLUSTER === "mainnet"
    ? "0x54a8e5f9c4CbA08F9943965859F6c34eAF03E26c"
    : CLUSTER === "testnet"
    ? "0x7bbcE28e64B3F8b84d876Ab298393c38ad7aac4C"
    : "0xC89Ce4735882C9F0f0FE26686c53074E09B0D550"
);
export const AVAX_NFT_BRIDGE_ADDRESS = getAddress(
  CLUSTER === "mainnet"
    ? "0xf7B6737Ca9c4e08aE573F75A97B73D7a813f5De5"
    : CLUSTER === "testnet"
    ? "0xD601BAf2EEE3C028344471684F6b27E789D9075D"
    : "0x26b4afb60d6c903165150c6f0aa14f8016be4aec"
);
export const AVAX_TOKEN_BRIDGE_ADDRESS = getAddress(
  CLUSTER === "mainnet"
    ? "0x0e082F06FF657D94310cB8cE8B0D9a04541d8052"
    : CLUSTER === "testnet"
    ? "0x61E44E506Ca5659E6c0bba9b678586fA2d729756"
    : "0x0290FB167208Af455bB137780163b7B7a9a10C16"
);
export const OASIS_BRIDGE_ADDRESS = getAddress(
  CLUSTER === "mainnet"
    ? "0xfE8cD454b4A1CA468B57D79c0cc77Ef5B6f64585"
    : CLUSTER === "testnet"
    ? "0xc1C338397ffA53a2Eb12A7038b4eeb34791F8aCb"
    : "0xC89Ce4735882C9F0f0FE26686c53074E09B0D550"
);
export const OASIS_NFT_BRIDGE_ADDRESS = getAddress(
  CLUSTER === "mainnet"
    ? "0x04952D522Ff217f40B5Ef3cbF659EcA7b952a6c1"
    : CLUSTER === "testnet"
    ? "0xC5c25B41AB0b797571620F5204Afa116A44c0ebA"
    : "0x26b4afb60d6c903165150c6f0aa14f8016be4aec"
);
export const OASIS_TOKEN_BRIDGE_ADDRESS = getAddress(
  CLUSTER === "mainnet"
    ? "0x5848C791e09901b40A9Ef749f2a6735b418d7564"
    : CLUSTER === "testnet"
    ? "0x88d8004A9BdbfD9D28090A02010C19897a29605c"
    : "0x0290FB167208Af455bB137780163b7B7a9a10C16"
);
export const SOL_BRIDGE_ADDRESS =
  CLUSTER === "mainnet"
    ? "worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth"
    : CLUSTER === "testnet"
    ? "3u8hJUVTA4jH1wYAyUur7FFZVQ8H635K3tSHHF4ssjQ5"
    : "Bridge1p5gheXUvJ6jGWGeCsgPKgnE3YgdGKRVCMY9o";
export const SOL_NFT_BRIDGE_ADDRESS =
  CLUSTER === "mainnet"
    ? "WnFt12ZrnzZrFZkt2xsNsaNWoQribnuQ5B5FrDbwDhD"
    : CLUSTER === "testnet"
    ? "2rHhojZ7hpu1zA91nvZmT8TqWWvMcKmmNBCr2mKTtMq4"
    : "NFTWqJR8YnRVqPDvTJrYuLrQDitTG5AScqbeghi4zSA";
export const SOL_TOKEN_BRIDGE_ADDRESS =
  CLUSTER === "mainnet"
    ? "wormDTUJ6AWPNvk59vGQbDvGJmqbDTdgWgAqcLBCgUb"
    : CLUSTER === "testnet"
    ? "DZnkkTmCiFWfYTfT41X3Rd1kDgozqzxWaHqsw6W4x2oe"
    : "B6RHG3mfcckmrYN1UhmJzyS1XX3fZKbkeUcpJe9Sy3FE";
export const ROPSTEN_ETH_BRIDGE_ADDRESS = getAddress(
  CLUSTER === "mainnet"
    ? "0x98f3c9e6E3fAce36bAAd05FE09d375Ef1464288B"
    : CLUSTER === "testnet"
    ? "0x210c5F5e2AF958B4defFe715Dc621b7a3BA888c5"
    : "0xC89Ce4735882C9F0f0FE26686c53074E09B0D550"
);
export const ROPSTEN_ETH_NFT_BRIDGE_ADDRESS = getAddress(
  CLUSTER === "mainnet"
    ? "0x6FFd7EdE62328b3Af38FCD61461Bbfc52F5651fE"
    : CLUSTER === "testnet"
    ? "0x2b048Da40f69c8dc386a56705915f8E966fe1eba"
    : "0x26b4afb60d6c903165150c6f0aa14f8016be4aec"
);
export const ROPSTEN_ETH_TOKEN_BRIDGE_ADDRESS = getAddress(
  CLUSTER === "mainnet"
    ? "0x3ee18B2214AFF97000D974cf647E7C347E8fa585"
    : CLUSTER === "testnet"
    ? "0xF174F9A837536C449321df1Ca093Bb96948D5386"
    : "0x0290FB167208Af455bB137780163b7B7a9a10C16"
);

export const SOL_CUSTODY_ADDRESS =
  "GugU1tP7doLeTw9hQP51xRJyS8Da1fWxuiy2rVrnMD2m";
export const SOL_NFT_CUSTODY_ADDRESS =
  "D63bhHo634eXSj4Jq3xgu2fjB5XKc8DFHzDY9iZk7fv1";
export const TERRA_TEST_TOKEN_ADDRESS =
  "terra13nkgqrfymug724h8pprpexqj9h629sa3ncw7sh";
export const TERRA_BRIDGE_ADDRESS =
  CLUSTER === "mainnet"
    ? "terra1dq03ugtd40zu9hcgdzrsq6z2z4hwhc9tqk2uy5"
    : CLUSTER === "testnet"
    ? "terra1pd65m0q9tl3v8znnz5f5ltsfegyzah7g42cx5v"
    : "terra18vd8fpwxzck93qlwghaj6arh4p7c5n896xzem5";
export const TERRA_TOKEN_BRIDGE_ADDRESS =
  CLUSTER === "mainnet"
    ? "terra10nmmwe8r3g99a9newtqa7a75xfgs2e8z87r2sf"
    : CLUSTER === "testnet"
    ? "terra1pseddrv0yfsn76u4zxrjmtf45kdlmalswdv39a"
    : "terra10pyejy66429refv3g35g2t7am0was7ya7kz2a4";

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
    : chainId === CHAIN_ID_ETHEREUM_ROPSTEN
    ? ROPSTEN_ETH_BRIDGE_ADDRESS
    : chainId === CHAIN_ID_AVAX
    ? AVAX_BRIDGE_ADDRESS
    : chainId === CHAIN_ID_OASIS
    ? OASIS_BRIDGE_ADDRESS
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
    : chainId === CHAIN_ID_ETHEREUM_ROPSTEN
    ? ROPSTEN_ETH_NFT_BRIDGE_ADDRESS
    : chainId === CHAIN_ID_AVAX
    ? AVAX_NFT_BRIDGE_ADDRESS
    : chainId === CHAIN_ID_OASIS
    ? OASIS_NFT_BRIDGE_ADDRESS
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
    : chainId === CHAIN_ID_ETHEREUM_ROPSTEN
    ? ROPSTEN_ETH_TOKEN_BRIDGE_ADDRESS
    : chainId === CHAIN_ID_AVAX
    ? AVAX_TOKEN_BRIDGE_ADDRESS
    : chainId === CHAIN_ID_OASIS
    ? OASIS_TOKEN_BRIDGE_ADDRESS
    : "";

export const COVALENT_API_KEY = process.env.REACT_APP_COVALENT_API_KEY
  ? process.env.REACT_APP_COVALENT_API_KEY
  : "";

export const COVALENT_ETHEREUM = 1; // Covalent only supports mainnet and Kovan
export const COVALENT_BSC = CLUSTER === "devnet" ? 56 : BSC_NETWORK_CHAIN_ID;
export const COVALENT_POLYGON =
  CLUSTER === "devnet" ? 137 : POLYGON_NETWORK_CHAIN_ID;
export const COVALENT_AVAX = CLUSTER === "devnet" ? 137 : AVAX_NETWORK_CHAIN_ID;
export const COVALENT_OASIS = CLUSTER === "devnet" ? null : null;
export const COVALENT_GET_TOKENS_URL = (
  chainId: ChainId,
  walletAddress: string,
  nft?: boolean,
  noNftMetadata?: boolean
) => {
  const chainNum =
    chainId === CHAIN_ID_ETH || chainId === CHAIN_ID_ETHEREUM_ROPSTEN
      ? COVALENT_ETHEREUM
      : chainId === CHAIN_ID_BSC
      ? COVALENT_BSC
      : chainId === CHAIN_ID_POLYGON
      ? COVALENT_POLYGON
      : chainId === CHAIN_ID_AVAX
      ? COVALENT_AVAX
      : chainId === CHAIN_ID_OASIS
      ? COVALENT_OASIS
      : "";
  // https://www.covalenthq.com/docs/api/#get-/v1/{chain_id}/address/{address}/balances_v2/
  return `https://api.covalenthq.com/v1/${chainNum}/address/${walletAddress}/balances_v2/?key=${COVALENT_API_KEY}${
    nft ? "&nft=true" : ""
  }${noNftMetadata ? "&no-nft-fetch=true" : ""}`;
};
export const TVL_URL =
  "https://europe-west3-wormhole-315720.cloudfunctions.net/mainnet-notionaltvl";
export const TERRA_SWAPRATE_URL =
  "https://fcd.terra.dev/v1/market/swaprate/uusd";

export const WETH_ADDRESS =
  CLUSTER === "mainnet"
    ? "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"
    : CLUSTER === "testnet"
    ? "0xb4fbf271143f4fbf7b91a5ded31805e42b2208d6"
    : "0xDDb64fE46a91D46ee29420539FC25FD07c5FEa3E";
export const WETH_DECIMALS = 18;

export const WBNB_ADDRESS =
  CLUSTER === "mainnet"
    ? "0xbb4CdB9CBd36B01bD1cBaEBF2De08d9173bc095c"
    : CLUSTER === "testnet"
    ? "0xae13d989dac2f0debff460ac112a837c89baa7cd"
    : "0xDDb64fE46a91D46ee29420539FC25FD07c5FEa3E";
export const WBNB_DECIMALS = 18;

export const WMATIC_ADDRESS =
  CLUSTER === "mainnet"
    ? "0x0d500b1d8e8ef31e21c99d1db9a6444d3adf1270"
    : CLUSTER === "testnet"
    ? "0x9c3c9283d3e44854697cd22d3faa240cfb032889"
    : "0xDDb64fE46a91D46ee29420539FC25FD07c5FEa3E";
export const WMATIC_DECIMALS = 18;

export const ROPSTEN_WETH_ADDRESS =
  CLUSTER === "mainnet"
    ? "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"
    : CLUSTER === "testnet"
    ? "0xc778417e063141139fce010982780140aa0cd5ab"
    : "0xDDb64fE46a91D46ee29420539FC25FD07c5FEa3E";
export const ROPSTEN_WETH_DECIMALS = 18;

export const WAVAX_ADDRESS =
  CLUSTER === "mainnet"
    ? "0xb31f66aa3c1e785363f0875a1b74e27b85fd66c7"
    : CLUSTER === "testnet"
    ? "0xd00ae08403b9bbb9124bb305c09058e32c39a48c"
    : "0xDDb64fE46a91D46ee29420539FC25FD07c5FEa3E";
export const WAVAX_DECIMALS = 18;

export const WROSE_ADDRESS =
  CLUSTER === "mainnet"
    ? "0x21C718C22D52d0F3a789b752D4c2fD5908a8A733"
    : CLUSTER === "testnet"
    ? "0x792296e2a15e6Ceb5f5039DecaE7A1f25b00B0B0"
    : "0xDDb64fE46a91D46ee29420539FC25FD07c5FEa3E";
export const WROSE_DECIMALS = 18;

export const WORMHOLE_V1_ETH_ADDRESS =
  CLUSTER === "mainnet"
    ? "0xf92cD566Ea4864356C5491c177A430C222d7e678"
    : CLUSTER === "testnet"
    ? "0xdae0Cba01eFc4bfEc1F7Fece73Fe8b8d2Eda65B0"
    : "0xf92cD566Ea4864356C5491c177A430C222d7e678"; //TODO something that doesn't explode in localhost
export const WORMHOLE_V1_SOLANA_ADDRESS =
  CLUSTER === "mainnet"
    ? "WormT3McKhFJ2RkiGpdw9GKvNCrB2aB54gb2uV9MfQC"
    : CLUSTER === "testnet"
    ? "BrdgiFmZN3BKkcY3danbPYyxPKwb8RhQzpM2VY5L97ED"
    : "";

export const TERRA_TOKEN_METADATA_URL =
  "https://assets.terra.money/cw20/tokens.json";

export const WORMHOLE_V1_MINT_AUTHORITY =
  CLUSTER === "mainnet"
    ? "9zyPU1mjgzaVyQsYwKJJ7AhVz5bgx5uc1NPABvAcUXsT"
    : CLUSTER === "testnet"
    ? "BJa7dq3bRP216zaTdw4cdcV71WkPc1HXvmnGeFVDi5DC"
    : "";

// hardcoded addresses for warnings
export const SOLANA_TOKENS_THAT_EXIST_ELSEWHERE = [
  "SRMuApVNdxXokk5GT7XD5cUUgXMBCoAz2LHeuAoKWRt", //  SRM
  "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", // USDC
  "kinXdEcpDQeHPEuQnqmUgtYykqKGVFq6CeVX5iAHJq6", //  KIN
  "CDJWUqTcYTVAKXAVXoQZFes5JUFc7owSeq7eMQcDSbo5", // renBTC
  "8wv2KAykQstNAj2oW6AHANGBiFKVFhvMiyyzzjhkmGvE", // renLUNA
  "G1a6jxYz3m8DVyMqYnuV7s86wD4fvuXYneWSpLJkmsXj", // renBCH
  "FKJvvVJ242tX7zFtzTmzqoA631LqHh4CdgcN8dcfFSju", // renDGB
  "ArUkYE2XDKzqy77PRRGjo4wREWwqk6RXTfM9NeqzPvjU", // renDOGE
  "E99CQ2gFMmbiyK2bwiaFNWUUmwz4r8k2CVEFxwuvQ7ue", // renZEC
  "De2bU64vsXKU9jq4bCjeDxNRGPn8nr3euaTK8jBYmD3J", // renFIL
  "Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB", // USDT
];
export const ETH_TOKENS_THAT_EXIST_ELSEWHERE = [
  getAddress("0x476c5E26a75bd202a9683ffD34359C0CC15be0fF"), // SRM
  getAddress("0x818fc6c2ec5986bc6e2cbf00939d90556ab12ce5"), // KIN
  getAddress("0xeb4c2781e4eba804ce9a9803c67d0893436bb27d"), // renBTC
  getAddress("0x52d87F22192131636F93c5AB18d0127Ea52CB641"), // renLUNA
  getAddress("0x459086f2376525bdceba5bdda135e4e9d3fef5bf"), // renBCH
  getAddress("0xe3cb486f3f5c639e98ccbaf57d95369375687f80"), // renDGB
  getAddress("0x3832d2F059E55934220881F831bE501D180671A7"), // renDOGE
  getAddress("0x1c5db575e2ff833e46a2e9864c22f4b22e0b37c2"), // renZEC
  getAddress("0xD5147bc8e386d91Cc5DBE72099DAC6C9b99276F5"), // renFIL
];
export const ETH_TOKENS_THAT_CAN_BE_SWAPPED_ON_SOLANA = [
  getAddress("0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"), // USDC
  getAddress("0xdac17f958d2ee523a2206206994597c13d831ec7"), // USDT
];
export const BSC_MARKET_WARNINGS = [
  getAddress(WBNB_ADDRESS),
  getAddress("0xe9e7cea3dedca5984780bafc599bd69add087d56"), // BUSD
  getAddress("0x8ac76a51cc950d9822d68b83fe1ad97b32cd580d"), // USDC
  getAddress("0x55d398326f99059ff775485246999027b3197955"), // BSC-USD
];

export const MIGRATION_PROGRAM_ADDRESS =
  CLUSTER === "mainnet"
    ? "whmRZnmyxdr2TkHXcZoFdtvNYRLQ5Jtbkf6ZbGkJjdk"
    : CLUSTER === "testnet"
    ? ""
    : "Ex9bCdVMSfx7EzB3pgSi2R4UHwJAXvTw18rBQm5YQ8gK";

export const MIGRATION_ASSET_MAP = new Map<string, string>(
  CLUSTER === "mainnet"
    ? [
        [
          // HUSD
          "BybpSTBoZHsmKnfxYG47GDhVPKrnEKX31CScShbrzUhX",
          "7VQo3HFLNH5QqGtM8eC3XQbPkJUu7nS9LeGWjerRh5Sw",
        ],
        [
          // BUSD
          "AJ1W9A9N9dEMdVyoDiam2rV44gnBm2csrPDP7xqcapgX",
          "33fsBLA8djQm82RpHmE3SuVrPGtZBWNYExsEUeKX1HXX",
        ],
        [
          // HBTC
          "8pBc4v9GAwCBNWPB5XKA93APexMGAS4qMr37vNke9Ref",
          "7dVH61ChzgmN9BwG4PkzwRP8PbYwPJ7ZPNF2vamKT2H8",
        ],
        [
          // DAI
          "FYpdBuyAHSbdaAyD1sKkxyLWbAP8uUW9h6uvdhK74ij1",
          "EjmyN6qEC1Tf1JxiG1ae7UTJhUxSwk1TCWNWqxWV4J6o",
        ],
        [
          // FRAX
          "8L8pDf3jutdpdr4m3np68CL9ZroLActrqwxi6s9Ah5xU",
          "FR87nWEUxVgerFGhZM8Y4AggKGLnaXswr1Pd8wZ4kZcp",
        ],
        [
          // USDK
          "2kycGCD8tJbrjJJqWN2Qz5ysN9iB4Bth3Uic4mSB7uak",
          "43m2ewFV5nDepieFjT9EmAQnc1HRtAF247RBpLGFem5F",
        ],
        // [
        //   // UST
        //   "CXLBjMMcwkc17GfJtBos6rQCo1ypeH6eDbB82Kby4MRm",
        //   "5Un6AdG9GBjxVhTSvvt2x6X6vtN1zrDxkkDpDcShnHfF",
        // ],
        [
          // UST
          "CXLBjMMcwkc17GfJtBos6rQCo1ypeH6eDbB82Kby4MRm",
          "9vMJfxuKxXBoEa7rM12mYLMwTacLMLDJqHozw96WQL8i",
        ],
        // [
        //   // Wrapped LUNA
        //   "2Xf2yAXJfg82sWwdLUo2x9mZXy6JCdszdMZkcF1Hf4KV",
        //   "EQTV1LW23Mgtjb5LXSg9NGw1J32oqTV4HCPmHCVSGmqD",
        // ],
        [
          // Wrapped LUNA
          "2Xf2yAXJfg82sWwdLUo2x9mZXy6JCdszdMZkcF1Hf4KV",
          "F6v4wfAdJB8D8p77bMXZgYt8TDKsYxLYxH5AFhUkYx9W",
        ],
        [
          // FTT
          "GbBWwtYTMPis4VHb8MrBbdibPhn28TSrLB53KvUmb7Gi",
          "EzfgjvkSwthhgHaceR3LnKXUoRkP6NUhfghdaHAj1tUv",
        ],
        [
          // SRM
          "2jXy799YnEcRXneFo2GEAB6SDRsAa767HpWmktRr1DaP",
          "xnorPhAzWXUczCP3KjU5yDxmKKZi5cSbxytQ1LgE3kG",
        ],
        [
          // FTT (Sollet)
          "AGFEad2et2ZJif9jaGpdMixQqvW5i81aBdvKe7PHNfz3",
          "EzfgjvkSwthhgHaceR3LnKXUoRkP6NUhfghdaHAj1tUv",
        ],
        [
          // WETH (Sollet)
          "2FPyTwcZLUg1MDrwsyoP4D6s1tM7hAkHYRjkNb5w6Pxk",
          "7vfCXTUXx5WJV5JADk17DUJ4ksgau7utNKj4b963voxs",
        ],
        [
          // UNI (Sollet)
          "DEhAasscXF4kEGxFgJ3bq4PpVGp5wyUxMRvn6TzGVHaw",
          "8FU95xFJhUUkyyCLU13HSzDLs7oC4QZdXQHL6SCeab36",
        ],
        [
          // HXRO (Sollet)
          "DJafV9qemGp7mLMEn5wrfqaFwxsbLgUsGVS16zKRk9kc",
          "HxhWkVpk5NS4Ltg5nij2G671CKXFRKPK8vy271Ub4uEK",
        ],
        [
          // ALEPH (Sollet)
          "CsZ5LZkDS7h9TDKjrbL7VAwQZ9nsRu8vJLhRYfmGaN8K",
          "3UCMiSnkcnkPE1pgQ5ggPCBv6dXgVUy16TmMUe1WpG9x",
        ],
        [
          // TOMOE (Sollet)
          "GXMvfY2jpQctDqZ9RoU3oWPhufKiCcFEfchvYumtX7jd",
          "46AiRdka3HYGkhV6r9gyS6Teo9cojfGXfK8oniALYMZx",
        ],
      ]
    : CLUSTER === "testnet"
    ? [
        [
          "orcarKHSqC5CDDsGbho8GKvwExejWHxTqGzXgcewB9L", //This is not actually a v1 asset
          "orcarKHSqC5CDDsGbho8GKvwExejWHxTqGzXgcewB9L",
        ],
      ]
    : [
        // [
        //   "2WDq7wSs9zYrpx2kbHDA4RUTRch2CCTP6ZWaH4GNfnQQ",
        //   "GcdupcwxkmVGM6s9F8bHSjNoznXAb3hRJTioABNYkn31",
        // ],
      ]
);

export const ETH_MIGRATION_ASSET_MAP = new Map<string, string>(
  CLUSTER === "mainnet"
    ? [
        [
          // LUNA
          getAddress("0xd2877702675e6cEb975b4A1dFf9fb7BAF4C91ea9"),
          getAddress("0xe76820F1DB773B1d62a3D22F47259705cC5fA4E6"),
        ],
        [
          // UST
          getAddress("0xa47c8bf37f92aBed4A126BDA807A7b7498661acD"),
          getAddress("0xF39C29d8f6851d87c40c83b61078EB7384f7Cb51"),
        ],
      ]
    : CLUSTER === "testnet"
    ? []
    : [
        // [
        //   "0x2D8BE6BF0baA74e0A907016679CaE9190e80dD0A",
        //   "0xFcCeD5E997E7fb1D0594518D3eD57245bB8ed17E",
        // ],
      ]
);

export const BSC_MIGRATION_ASSET_MAP = new Map<string, string>(
  CLUSTER === "mainnet"
    ? [
        [
          // LUNA
          getAddress("0xECCF35F941Ab67FfcAA9A1265C2fF88865caA005"),
          getAddress("0x355A116ef1Cf566B12Ef3a8e409A64e303c53740"),
        ],
        [
          // UST
          getAddress("0x23396cF899Ca06c4472205fC903bDB4de249D6fC"),
          getAddress("0x0F98AB919D04a291838B3b075c57181057D4CF75"),
        ],
        [
          // ORION
          getAddress("0x5530ec23f4ee1521182bd158c09f4ca7efec1ef0"),
          getAddress("0x084fa354e65b521e6fb9d1602549cf8693cdb4f8"),
        ],
      ]
    : CLUSTER === "testnet"
    ? []
    : [
        // [
        //   "0x2D8BE6BF0baA74e0A907016679CaE9190e80dD0A",
        //   "0xFcCeD5E997E7fb1D0594518D3eD57245bB8ed17E",
        // ],
      ]
);

export const getMigrationAssetMap = (chainId: ChainId) => {
  if (chainId === CHAIN_ID_BSC) {
    return BSC_MIGRATION_ASSET_MAP;
  } else if (chainId === CHAIN_ID_ETH) {
    return ETH_MIGRATION_ASSET_MAP;
  } else if (chainId === CHAIN_ID_SOLANA) {
    return MIGRATION_ASSET_MAP;
  } else {
    return new Map<string, string>();
  }
};

export const SUPPORTED_TERRA_TOKENS = ["uluna", "uusd"];
export const TERRA_DEFAULT_FEE_DENOM = SUPPORTED_TERRA_TOKENS[0];

export const TERRA_FCD_BASE =
  CLUSTER === "mainnet"
    ? "https://fcd.terra.dev"
    : CLUSTER === "testnet"
    ? "https://bombay-fcd.terra.dev"
    : "http://localhost:3060";
export const TERRA_GAS_PRICES_URL = `${TERRA_FCD_BASE}/v1/txs/gas_prices`;

export const TOTAL_TRANSACTIONS_WORMHOLE = `https://europe-west3-wormhole-315720.cloudfunctions.net/mainnet-totals?groupBy=address`;

export const RECENT_TRANSACTIONS_WORMHOLE = `https://europe-west3-wormhole-315720.cloudfunctions.net/mainnet-recent?groupBy=address&numRows=2`;

export const VAA_EMITTER_ADDRESSES = [
  `${CHAIN_ID_SOLANA}:ec7372995d5cc8732397fb0ad35c0121e0eaa90d26f828a534cab54391b3a4f5`, //SOLANA TOKEN
  `${CHAIN_ID_SOLANA}:0def15a24423e1edd1a5ab16f557b9060303ddbab8c803d2ee48f4b78a1cfd6b`, //SOLAN NFT
  `${CHAIN_ID_ETH}:0000000000000000000000003ee18b2214aff97000d974cf647e7c347e8fa585`, //ETH token
  `${CHAIN_ID_ETH}:0000000000000000000000006ffd7ede62328b3af38fcd61461bbfc52f5651fe`, //ETH NFT
  `${CHAIN_ID_TERRA}:0000000000000000000000007cf7b764e38a0a5e967972c1df77d432510564e2`, //terra
  `${CHAIN_ID_BSC}:000000000000000000000000b6f6d86a8f9879a9c87f643768d9efc38c1da6e7`, //bsc
  `${CHAIN_ID_BSC}:0000000000000000000000005a58505a96d1dbf8df91cb21b54419fc36e93fde`, //bsc nft
  `${CHAIN_ID_POLYGON}:0000000000000000000000005a58505a96d1dbf8df91cb21b54419fc36e93fde`, //Polygon
  `${CHAIN_ID_POLYGON}:00000000000000000000000090bbd86a6fe93d3bc3ed6335935447e75fab7fcf`, //Polygon nft
  `${CHAIN_ID_AVAX}:0000000000000000000000000e082f06ff657d94310cb8ce8b0d9a04541d8052`, //AVAX
  `${CHAIN_ID_AVAX}:000000000000000000000000f7b6737ca9c4e08ae573f75a97b73d7a813f5de5`, //AVAX nft
  `${CHAIN_ID_OASIS}:0000000000000000000000005848c791e09901b40a9ef749f2a6735b418d7564`, //OASIS
  `${CHAIN_ID_OASIS}:00000000000000000000000004952D522Ff217f40B5Ef3cbF659EcA7b952a6c1`, //OASIS nft
];

export const WORMHOLE_EXPLORER_BASE = "https://wormholenetwork.com/en/explorer";

export type MultiChainInfo = {
  [key in ChainId]: { [address: string]: string };
};
export const MULTI_CHAIN_TOKENS: MultiChainInfo =
  //EVM chains should format the addresses to all lowercase
  CLUSTER === "mainnet"
    ? ({
        [CHAIN_ID_SOLANA]: {
          EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v: "USDC",
          Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB: "USDT",
        },
        [CHAIN_ID_ETH]: {
          "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48": "USDC",
          "0xdac17f958d2ee523a2206206994597c13d831ec7": "USDT",
        },
        [CHAIN_ID_TERRA]: {},
        [CHAIN_ID_BSC]: {
          "0x8ac76a51cc950d9822d68b83fe1ad97b32cd580d": "USDC",
          "0x55d398326f99059ff775485246999027b3197955": "USDT",
        },
        [CHAIN_ID_POLYGON]: {
          "0x2791bca1f2de4661ed88a30c99a7a9449aa84174": "USDC",
          "0xc2132d05d31c914a87c6611c10748aeb04b58e8f": "USDT",
        },
      } as MultiChainInfo)
    : ({
        [CHAIN_ID_SOLANA]: {
          "2WDq7wSs9zYrpx2kbHDA4RUTRch2CCTP6ZWaH4GNfnQQ": "SOLT",
        },
        [CHAIN_ID_ETH]: {},
        [CHAIN_ID_TERRA]: {},
        [CHAIN_ID_BSC]: {},
        [CHAIN_ID_POLYGON]: {},
      } as MultiChainInfo);

export const AVAILABLE_MARKETS_URL =
  "https://docs.wormholenetwork.com/wormhole/overview-liquid-markets";

export const SOLANA_SYSTEM_PROGRAM_ADDRESS = "11111111111111111111111111111111";
export const FEATURED_MARKETS_JSON_URL =
  "https://raw.githubusercontent.com/certusone/wormhole-token-list/main/src/markets.json";

export const logoOverrides = new Map<string, string>([
  [
    "0x727f064a78dc734d33eec18d5370aef32ffd46e4",
    "https://orion.money/assets/ORION-LOGO-2.1-GREEN@256x256.png",
  ],
]);

export const getHowToAddTokensToWalletUrl = (chainId: ChainId) => {
  if (isEVMChain(chainId)) {
    return "https://docs.wormholenetwork.com/wormhole/video-tutorial-how-to-manually-add-tokens-to-your-wallet#1.-metamask-ethereum-polygon-and-bsc";
  } else if (chainId === CHAIN_ID_TERRA) {
    return "https://docs.wormholenetwork.com/wormhole/video-tutorial-how-to-manually-add-tokens-to-your-wallet#2.-terra-station";
  }
  return "";
};

export const getHowToAddToTokenListUrl = (chainId: ChainId) => {
  if (chainId === CHAIN_ID_SOLANA) {
    return "https://github.com/solana-labs/token-list";
  } else if (chainId === CHAIN_ID_TERRA) {
    return "https://github.com/terra-money/assets";
  }
  return "";
};

export const SOLANA_TOKEN_METADATA_PROGRAM_URL =
  "https://github.com/metaplex-foundation/metaplex/tree/master/rust/token-metadata/program";
export const MAX_VAA_UPLOAD_RETRIES_SOLANA = 5;

export const POLYGON_TERRA_WRAPPED_TOKENS = [
  "0x692597b009d13c4049a947cab2239b7d6517875f", // Wrapped UST Token
  "0x24834bbec7e39ef42f4a75eaf8e5b6486d3f0e57", // Wrapped LUNA Token
];

export const JUPITER_SWAP_BASE_URL = "https://jup.ag/swap";
