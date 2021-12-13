import {
  ChainId,
  CHAIN_ID_SOLANA,
  CHAIN_ID_TERRA,
  isEVMChain,
} from "@certusone/wormhole-sdk";
import { TokenInfo } from "@solana/spl-token-registry";
import { useMemo } from "react";
import { DataWrapper, getEmptyDataWrapper } from "../store/helpers";
import { logoOverrides } from "../utils/consts";
import { Metadata } from "../utils/metaplex";
import useEvmMetadata, { EvmMetadata } from "./useEvmMetadata";
import useMetaplexData from "./useMetaplexData";
import useSolanaTokenMap from "./useSolanaTokenMap";
import useTerraMetadata, { TerraMetadata } from "./useTerraMetadata";
import useTerraTokenMap, { TerraTokenMap } from "./useTerraTokenMap";

export type GenericMetadata = {
  symbol?: string;
  logo?: string;
  tokenName?: string;
  decimals?: number;
  //TODO more items
  raw?: any;
};

const constructSolanaMetadata = (
  addresses: string[],
  solanaTokenMap: DataWrapper<TokenInfo[]>,
  metaplexData: DataWrapper<Map<string, Metadata | undefined> | undefined>
) => {
  const isFetching = solanaTokenMap.isFetching || metaplexData?.isFetching;
  const error = solanaTokenMap.error || metaplexData?.isFetching;
  const receivedAt = solanaTokenMap.receivedAt && metaplexData?.receivedAt;
  const data = new Map<string, GenericMetadata>();
  addresses.forEach((address) => {
    const metaplex = metaplexData?.data?.get(address);
    const tokenInfo = solanaTokenMap.data?.find((x) => x.address === address);
    //Both this and the token picker, at present, give priority to the tokenmap
    const obj = {
      symbol: metaplex?.data?.symbol || tokenInfo?.symbol || undefined,
      logo: tokenInfo?.logoURI || undefined, //TODO is URI on metaplex actually the logo? If not, where is it?
      tokenName: metaplex?.data?.name || tokenInfo?.name || undefined,
      decimals: tokenInfo?.decimals || undefined, //TODO decimals are actually on the mint, not the metaplex account.
      raw: metaplex,
    };
    data.set(address, obj);
  });

  return {
    isFetching,
    error,
    receivedAt,
    data,
  };
};

const constructTerraMetadata = (
  addresses: string[],
  tokenMap: DataWrapper<TerraTokenMap>,
  terraMetadata: DataWrapper<Map<string, TerraMetadata>>
) => {
  const isFetching = tokenMap.isFetching || terraMetadata.isFetching;
  const error = tokenMap.error || terraMetadata.error;
  const receivedAt = tokenMap.receivedAt && terraMetadata.receivedAt;
  const data = new Map<string, GenericMetadata>();
  addresses.forEach((address) => {
    const metadata = terraMetadata.data?.get(address);
    const tokenInfo = tokenMap.data?.mainnet[address];
    const obj = {
      symbol: tokenInfo?.symbol || metadata?.symbol || undefined,
      logo: tokenInfo?.icon || metadata?.logo || undefined,
      tokenName: tokenInfo?.name || metadata?.tokenName || undefined,
      decimals: metadata?.decimals || undefined,
    };
    data.set(address, obj);
  });

  return {
    isFetching,
    error,
    receivedAt,
    data,
  };
};

const constructEthMetadata = (
  addresses: string[],
  metadataMap: DataWrapper<Map<string, EvmMetadata> | null>
) => {
  const isFetching = metadataMap.isFetching;
  const error = metadataMap.error;
  const receivedAt = metadataMap.receivedAt;
  const data = new Map<string, GenericMetadata>();
  addresses.forEach((address) => {
    const meta = metadataMap.data?.get(address);
    const obj = {
      symbol: meta?.symbol || undefined,
      logo: logoOverrides.get(address) || meta?.logo || undefined,
      tokenName: meta?.tokenName || undefined,
      decimals: meta?.decimals,
    };
    data.set(address, obj);
  });

  return {
    isFetching,
    error,
    receivedAt,
    data,
  };
};

export default function useMetadata(
  chainId: ChainId,
  addresses: string[]
): DataWrapper<Map<string, GenericMetadata>> {
  const terraTokenMap = useTerraTokenMap(chainId === CHAIN_ID_TERRA);
  const solanaTokenMap = useSolanaTokenMap();

  const solanaAddresses = useMemo(() => {
    return chainId === CHAIN_ID_SOLANA ? addresses : [];
  }, [chainId, addresses]);
  const terraAddresses = useMemo(() => {
    return chainId === CHAIN_ID_TERRA ? addresses : [];
  }, [chainId, addresses]);
  const ethereumAddresses = useMemo(() => {
    return isEVMChain(chainId) ? addresses : [];
  }, [chainId, addresses]);

  const metaplexData = useMetaplexData(solanaAddresses);
  const terraMetadata = useTerraMetadata(terraAddresses);
  const ethMetadata = useEvmMetadata(ethereumAddresses, chainId);

  const output: DataWrapper<Map<string, GenericMetadata>> = useMemo(
    () =>
      chainId === CHAIN_ID_SOLANA
        ? constructSolanaMetadata(solanaAddresses, solanaTokenMap, metaplexData)
        : isEVMChain(chainId)
        ? constructEthMetadata(ethereumAddresses, ethMetadata)
        : chainId === CHAIN_ID_TERRA
        ? constructTerraMetadata(terraAddresses, terraTokenMap, terraMetadata)
        : getEmptyDataWrapper(),
    [
      chainId,
      solanaAddresses,
      solanaTokenMap,
      metaplexData,
      ethereumAddresses,
      ethMetadata,
      terraAddresses,
      terraMetadata,
      terraTokenMap,
    ]
  );

  return output;
}
