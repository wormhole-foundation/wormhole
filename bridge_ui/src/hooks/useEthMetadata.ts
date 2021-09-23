import { CHAIN_ID_ETH } from "@certusone/wormhole-sdk";
import { ethers } from "@certusone/wormhole-sdk/node_modules/ethers";
import { useEffect, useMemo, useState } from "react";
import {
  Provider,
  useEthereumProvider,
} from "../contexts/EthereumProviderContext";
import { DataWrapper } from "../store/helpers";
import useIsWalletReady from "./useIsWalletReady";

export type EthMetadata = {
  symbol?: string;
  logo?: string;
  tokenName?: string;
  decimals?: number;
};

const ERC20_BASIC_ABI = [
  "function name() view returns (string name)",
  "function symbol() view returns (string symbol)",
  "function decimals() view returns (uint8 decimals)",
];

const handleError = () => {
  return undefined;
};

const fetchSingleMetadata = async (
  address: string,
  provider: Provider
): Promise<EthMetadata> => {
  const contract = new ethers.Contract(address, ERC20_BASIC_ABI, provider);
  const [name, symbol, decimals] = await Promise.all([
    contract.name().catch(handleError),
    contract.symbol().catch(handleError),
    contract.decimals().catch(handleError),
  ]);
  return { tokenName: name, symbol, decimals };
};

const fetchEthMetadata = async (addresses: string[], provider: Provider) => {
  const promises: Promise<EthMetadata>[] = [];
  addresses.forEach((address) => {
    promises.push(fetchSingleMetadata(address, provider));
  });
  const resultsArray = await Promise.all(promises);
  const output = new Map<string, EthMetadata>();
  addresses.forEach((address, index) => {
    output.set(address, resultsArray[index]);
  });

  return output;
};

function useEthMetadata(
  addresses: string[]
): DataWrapper<Map<string, EthMetadata>> {
  const { isReady } = useIsWalletReady(CHAIN_ID_ETH);
  const { provider } = useEthereumProvider();

  const [isFetching, setIsFetching] = useState(false);
  const [error, setError] = useState("");
  const [data, setData] = useState<Map<string, EthMetadata> | null>(null);

  useEffect(() => {
    let cancelled = false;
    if (addresses.length && provider && isReady) {
      setIsFetching(true);
      setError("");
      setData(null);
      fetchEthMetadata(addresses, provider).then(
        (results) => {
          if (!cancelled) {
            setData(results);
            setIsFetching(false);
          }
        },
        () => {
          if (!cancelled) {
            setError("Could not retrieve contract metadata");
            setIsFetching(false);
          }
        }
      );
    }
    return () => {
      cancelled = true;
    };
  }, [addresses, provider, isReady]);

  return useMemo(
    () => ({
      data,
      isFetching,
      error,
      receivedAt: null,
    }),
    [data, isFetching, error]
  );
}

export default useEthMetadata;
