import { MapLevel, zip } from "../utils";
import { constMap } from "../utils/mapping";

const chainsAndChainIdEntries = [
  //Unlike the old sdk, we are not including an "Unset" chain with chainId 0 here because:
  //  * no other types would be associated with it (such as contracts or a platform)
  //  * avoids awkward "chain but not 'Unset'" checks
  //  * "off" is not a TV channel either
  //Instead we'll use `null` for chain and 0 as the chainId where appropriate (e.g. governance VAAs)
  ["Solana", 1],
  ["Ethereum", 2],
  ["Terra", 3],
  ["Bsc", 4],
  ["Polygon", 5],
  ["Avalanche", 6],
  ["Oasis", 7],
  ["Algorand", 8],
  ["Aurora", 9],
  ["Fantom", 10],
  ["Karura", 11],
  ["Acala", 12],
  ["Klaytn", 13],
  ["Celo", 14],
  ["Near", 15],
  ["Moonbeam", 16],
  ["Neon", 17],
  ["Terra2", 18],
  ["Injective", 19],
  ["Osmosis", 20],
  ["Sui", 21],
  ["Aptos", 22],
  ["Arbitrum", 23],
  ["Optimism", 24],
  ["Gnosis", 25],
  ["Pythnet", 26],
  ["Xpla", 28],
  ["Btc", 29],
  ["Base", 30],
  ["Sei", 32],
  ["Rootstock", 33],
  ["Wormchain", 3104],
  ["Cosmoshub", 4000],
  ["Evmos", 4001],
  ["Kujira", 4002],
  // holy cow, how ugly of a hack is that?! - a chainId that's exclusive to a testnet!
  ["Sepolia", 10002],
] as const satisfies MapLevel<string, number>;

export const [chains, chainIds] = zip(chainsAndChainIdEntries);
export type Chain = (typeof chains)[number];
export type ChainId = (typeof chainIds)[number];

export const chainToChainId = constMap(chainsAndChainIdEntries);
export const chainIdToChain = constMap(chainsAndChainIdEntries, [1, 0]);

export const isChain = (chain: string): chain is Chain => chainToChainId.has(chain);
export const isChainId = (chainId: number): chainId is ChainId => chainIdToChain.has(chainId);

export function assertChainId(chainId: number): asserts chainId is ChainId {
  if (!isChainId(chainId)) throw Error(`Unknown Wormhole chain id: ${chainId}`);
}

export function assertChain(chain: string): asserts chain is Chain {
  if (!isChain(chain)) throw Error(`Unknown Wormhole chain: ${chain}`);
}

//safe assertion that allows chaining
export const asChainId = (chainId: number): ChainId => {
  assertChainId(chainId);
  return chainId;
};


export const toChainId = (chain: number | string): ChainId => {
  switch (typeof chain) {
    case "string":
      if (isChain(chain)) return chainToChainId(chain);
      break;
    case "number":
      if (isChainId(chain)) return chain;
      break;
  }
  throw Error(`Cannot convert to ChainId: ${chain}`);
};

export const toChain = (chain: number | string | bigint): Chain => {
  switch (typeof chain) {
    case "string":
      if (isChain(chain)) return chain;
      break;
    case "number":
      if (isChainId(chain)) return chainIdToChain(chain);
      break;
    case "bigint":
      if (isChainId(Number(chain))) return chainIdToChain(Number(chain) as ChainId);
      break;
  }
  throw Error(`Cannot convert to Chain: ${chain}`);
};
