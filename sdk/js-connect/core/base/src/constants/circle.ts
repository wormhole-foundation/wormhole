import { Network } from "./networks";
import { Chain, ChainName } from "./chains";
import { zip, constMap, RoArray } from "../utils";

const circleAPIs = [
  ["Mainnet", "https://iris-api.circle.com/v1/attestations"],
  ["Testnet", "https://iris-api-sandbox.circle.com/v1/attestations"],
] as const satisfies RoArray<readonly [Network, string]>;

// https://developers.circle.com/stablecoin/docs/cctp-technical-reference#domain-list
const circleDomains = [
  ["Ethereum", 0],
  ["Avalanche", 1],
  ["Optimism", 2],
  ["Arbitrum", 3],
  ["Base", 6],
] as const satisfies RoArray<readonly [ChainName, number]>;

const usdcContracts = [
  [
    "Mainnet",
    [
      ["Ethereum", "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"],
      ["Avalanche", "0xb97ef9ef8734c71904d8002f8b6bc66dd9c48a6e"],
      ["Arbitrum", "0xaf88d065e77c8cC2239327C5EDb3A432268e5831"],
      ["Optimism", "0x179522635726710dd7d2035a81d856de4aa7836c"],
      ["Base", "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913"],
    ],
  ],
  [
    "Testnet",
    [
      ["Avalanche", "0x5425890298aed601595a70AB815c96711a31Bc65"],
      ["Arbitrum", "0xfd064A18f3BF249cf1f87FC203E90D8f650f2d63"],
      ["Ethereum", "0x07865c6e87b9f70255377e024ace6630c1eaa37f"],
      ["Optimism", "0xe05606174bac4A6364B31bd0eCA4bf4dD368f8C6"],
      ["Base", "0xf175520c52418dfe19c8098071a252da48cd1c19"],
    ],
  ],
] as const satisfies RoArray<readonly [Network, RoArray<readonly [ChainName, string]>]>;

export const [circleChains, circleChainIds] = zip(circleDomains);
export type CircleChainName = (typeof circleChains)[number];
export type CircleChainId = (typeof circleChainIds)[number];

export const [circleNetworks, _] = zip(usdcContracts);
export type CircleNetwork = (typeof circleNetworks)[number];

export const circleChainId = constMap(circleDomains);
export const circleChainIdToChainName = constMap(circleDomains, [1, 0]);
export const circleAPI = constMap(circleAPIs);

export const usdcContract = constMap(usdcContracts);

export const isCircleChain = (
  chain: string | ChainName | CircleChainName,
): chain is CircleChainName => circleChainId.has(chain);

export const isCircleChainId = (chainId: number): chainId is CircleChainId =>
  circleChainIdToChainName.has(chainId);

export const isCircleSupported = (
  network: Network,
  chain: string | ChainName | CircleChainName,
): network is CircleNetwork => usdcContract.has(network, chain);

export function assertCircleChainId(chainId: number): asserts chainId is CircleChainId {
  if (!isCircleChainId(chainId)) throw Error(`Unknown Circle chain id: ${chainId}`);
}

export function assertCircleChain(chain: string): asserts chain is CircleChainName {
  if (!isCircleChain(chain)) throw Error(`Unknown Circle chain: ${chain}`);
}

//safe assertion that allows chaining
export const asCircleChainId = (chainId: number): CircleChainId => {
  assertCircleChainId(chainId);
  return chainId;
};

export const toCircleChainId = (chain: number | bigint | string | Chain): CircleChainId => {
  switch (typeof chain) {
    case "string":
      if (isCircleChain(chain)) return circleChainId(chain);
      break;
    case "number":
      if (isCircleChainId(chain)) return chain;
      break;
    case "bigint":
      const ci = Number(chain);
      if (isCircleChainId(ci)) return ci;
      break;
  }
  throw Error(`Cannot convert to ChainId: ${chain}`);
};

export const toCircleChainName = (chain: number | string | Chain | bigint): ChainName => {
  switch (typeof chain) {
    case "string":
      if (isCircleChain(chain)) return chain;
      break;
    case "number":
      if (isCircleChainId(chain)) return circleChainIdToChainName(chain);
      break;
    case "bigint":
      const cid = Number(chain);
      if (isCircleChainId(cid)) return circleChainIdToChainName(cid);
      break;
  }
  throw Error(`Cannot convert to ChainName: ${chain}`);
};
