import { Algodv2 } from "algosdk";
import { useEffect, useMemo, useState } from "react";
import { DataWrapper } from "../store/helpers";
import { ALGORAND_HOST, ALGO_DECIMALS } from "../utils/consts";

export type AlgoMetadata = {
  symbol?: string;
  tokenName?: string;
  decimals: number;
};

export const fetchSingleMetadata = async (
  address: string,
  algodClient: Algodv2
): Promise<AlgoMetadata> => {
  const assetId = parseInt(address);
  if (assetId === 0) {
    return {
      tokenName: "Algo",
      symbol: "ALGO",
      decimals: ALGO_DECIMALS,
    };
  }
  const assetInfo = await algodClient.getAssetByID(assetId).do();
  return {
    tokenName: assetInfo.params.name,
    symbol: assetInfo.params["unit-name"],
    decimals: assetInfo.params.decimals,
  };
};

const fetchAlgoMetadata = async (addresses: string[]) => {
  const algodClient = new Algodv2(
    ALGORAND_HOST.algodToken,
    ALGORAND_HOST.algodServer,
    ALGORAND_HOST.algodPort
  );
  const promises: Promise<AlgoMetadata>[] = [];
  addresses.forEach((address) => {
    promises.push(fetchSingleMetadata(address, algodClient));
  });
  const resultsArray = await Promise.all(promises);
  const output = new Map<string, AlgoMetadata>();
  addresses.forEach((address, index) => {
    output.set(address, resultsArray[index]);
  });

  return output;
};

function useAlgoMetadata(
  addresses: string[]
): DataWrapper<Map<string, AlgoMetadata>> {
  const [isFetching, setIsFetching] = useState(false);
  const [error, setError] = useState("");
  const [data, setData] = useState<Map<string, AlgoMetadata> | null>(null);

  useEffect(() => {
    let cancelled = false;
    if (addresses.length) {
      setIsFetching(true);
      setError("");
      setData(null);
      fetchAlgoMetadata(addresses).then(
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
  }, [addresses]);

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

export default useAlgoMetadata;
