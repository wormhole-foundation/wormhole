import { TerraChainId } from "@certusone/wormhole-sdk";
import { LCDClient } from "@terra-money/terra.js";
import { useLayoutEffect, useMemo, useState } from "react";
import { DataWrapper } from "../store/helpers";
import { getTerraConfig } from "../utils/consts";

export type TerraMetadata = {
  symbol?: string;
  logo?: string;
  tokenName?: string;
  decimals?: number;
};

const fetchSingleMetadata = async (address: string, lcd: LCDClient) =>
  lcd.wasm
    .contractQuery(address, {
      token_info: {},
    })
    .then(
      ({ symbol, name: tokenName, decimals }: any) =>
        ({
          symbol,
          tokenName,
          decimals,
        } as TerraMetadata)
    );

const fetchTerraMetadata = async (addresses: string[], chainId: TerraChainId) => {
  const lcd = new LCDClient(getTerraConfig(chainId));
  const promises: Promise<TerraMetadata>[] = [];
  addresses.forEach((address) => {
    promises.push(fetchSingleMetadata(address, lcd));
  });
  const resultsArray = await Promise.all(promises);
  const output = new Map<string, TerraMetadata>();
  addresses.forEach((address, index) => {
    output.set(address, resultsArray[index]);
  });

  return output;
};

const useTerraMetadata = (
  addresses: string[],
  chainId: TerraChainId
): DataWrapper<Map<string, TerraMetadata>> => {
  const [isFetching, setIsFetching] = useState(false);
  const [error, setError] = useState("");
  const [data, setData] = useState<Map<string, TerraMetadata> | null>(null);

  useLayoutEffect(() => {
    let cancelled = false;
    if (addresses.length) {
      setIsFetching(true);
      setError("");
      setData(null);
      fetchTerraMetadata(addresses, chainId).then(
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
  }, [addresses, chainId]);

  return useMemo(
    () => ({
      data,
      isFetching,
      error,
      receivedAt: null,
    }),
    [data, isFetching, error]
  );
};

export default useTerraMetadata;
