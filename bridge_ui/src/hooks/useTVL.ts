import {
  ChainId,
  CHAIN_ID_BSC,
  CHAIN_ID_ETH,
  CHAIN_ID_SOLANA,
  CHAIN_ID_TERRA,
} from "@certusone/wormhole-sdk";
import { formatUnits } from "@ethersproject/units";
import { TOKEN_PROGRAM_ID } from "@solana/spl-token";
import {
  AccountInfo,
  Connection,
  ParsedAccountData,
  PublicKey,
} from "@solana/web3.js";
import axios from "axios";
import { useEffect, useMemo, useState } from "react";
import { DataWrapper } from "../store/helpers";
import {
  BSC_TOKEN_BRIDGE_ADDRESS,
  CHAINS_BY_ID,
  COVALENT_GET_TOKENS_URL,
  ETH_TOKEN_BRIDGE_ADDRESS,
  SOLANA_HOST,
  SOL_CUSTODY_ADDRESS,
  TERRA_SWAPRATE_URL,
  TERRA_TOKEN_BRIDGE_ADDRESS,
} from "../utils/consts";
import {
  formatNativeDenom,
  getNativeTerraIcon,
  NATIVE_TERRA_DECIMALS,
} from "../utils/terra";
import useMetadata, { GenericMetadata } from "./useMetadata";
import useTerraNativeBalances from "./useTerraNativeBalances";

export type TVL = {
  logo?: string;
  symbol?: string;
  name?: string;
  amount: string;
  totalValue?: number;
  quotePrice?: number;
  assetAddress: string;
  originChainId: ChainId;
  originChain: string;
};

const calcEvmTVL = (covalentReport: any, chainId: ChainId): TVL[] => {
  const output: TVL[] = [];
  if (!covalentReport?.data?.items?.length) {
    return [];
  }

  covalentReport.data.items.forEach((item: any) => {
    if (item.balance > 0 && item.contract_address) {
      output.push({
        logo: item.logo_url || undefined,
        symbol: item.contract_ticker_symbol || undefined,
        name: item.contract_name || undefined,
        amount: formatUnits(item.balance, item.contract_decimals),
        totalValue: item.quote,
        quotePrice: item.quote_rate,
        assetAddress: item.contract_address,
        originChainId: chainId,
        originChain: CHAINS_BY_ID[chainId].name,
      });
    }
  });

  return output;
};
const calcSolanaTVL = (
  accounts:
    | { pubkey: PublicKey; account: AccountInfo<ParsedAccountData> }[]
    | undefined,
  metaData: DataWrapper<Map<string, GenericMetadata>>
) => {
  const output: TVL[] = [];
  if (
    !accounts ||
    !accounts.length ||
    metaData.isFetching ||
    metaData.error ||
    !metaData.data
  ) {
    return output;
  }

  accounts.forEach((item) => {
    const genericMetadata = metaData.data?.get(
      item.account.data.parsed?.info?.mint?.toString()
    );
    output.push({
      logo: genericMetadata?.logo || undefined,
      symbol: genericMetadata?.symbol || undefined,
      name: genericMetadata?.tokenName || undefined,
      amount: item.account.data.parsed?.info?.tokenAmount?.uiAmount || "0", //Should always be defined.
      totalValue: undefined,
      quotePrice: undefined,
      assetAddress: item.account.data.parsed?.info?.mint?.toString(),
      originChainId: CHAIN_ID_SOLANA,
      originChain: "Solana",
    });
  });

  return output;
};

const useTerraTVL = () => {
  const { isLoading: isTerraNativeLoading, balances: terraNativeBalances } =
    useTerraNativeBalances(TERRA_TOKEN_BRIDGE_ADDRESS);
  const [terraSwaprates, setTerraSwaprates] = useState<any[]>([]);
  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const result = await axios.get(TERRA_SWAPRATE_URL);
        if (!cancelled && result && result.data) {
          setTerraSwaprates(result.data);
        }
      } catch (e) {}
    })();
    return () => {
      cancelled = true;
    };
  }, []);
  const terraTVL = useMemo(() => {
    const arr: TVL[] = [];
    if (terraNativeBalances) {
      const denoms = Object.keys(terraNativeBalances);
      denoms.forEach((denom) => {
        const amount = formatUnits(
          terraNativeBalances[denom],
          NATIVE_TERRA_DECIMALS
        );
        const symbol = formatNativeDenom(denom);
        let matchingSwap = undefined;
        let quotePrice = 0;
        let totalValue = 0;
        try {
          matchingSwap = terraSwaprates.find((swap) => swap.denom === denom);
          quotePrice =
            denom === "uusd"
              ? 1
              : matchingSwap
              ? 1 / Number(matchingSwap.swaprate)
              : 0;
          totalValue =
            denom === "uusd"
              ? Number(
                  formatUnits(terraNativeBalances[denom], NATIVE_TERRA_DECIMALS)
                )
              : matchingSwap
              ? Number(amount) / Number(matchingSwap.swaprate)
              : 0;
        } catch (e) {}
        arr.push({
          amount,
          assetAddress: denom,
          originChain: CHAINS_BY_ID[CHAIN_ID_TERRA].name,
          originChainId: CHAIN_ID_TERRA,
          quotePrice,
          totalValue,
          logo: getNativeTerraIcon(symbol),
          symbol,
        });
      });
    }
    return arr;
  }, [terraNativeBalances, terraSwaprates]);
  return useMemo(
    () => ({ terraTVL, isLoading: isTerraNativeLoading }),
    [isTerraNativeLoading, terraTVL]
  );
};

const useTVL = (): DataWrapper<TVL[]> => {
  const [ethCovalentData, setEthCovalentData] = useState(undefined);
  const [ethCovalentIsLoading, setEthCovalentIsLoading] = useState(false);
  const [ethCovalentError, setEthCovalentError] = useState("");

  const [bscCovalentData, setBscCovalentData] = useState(undefined);
  const [bscCovalentIsLoading, setBscCovalentIsLoading] = useState(false);
  const [bscCovalentError, setBscCovalentError] = useState("");

  const [solanaCustodyTokens, setSolanaCustodyTokens] = useState<
    { pubkey: PublicKey; account: AccountInfo<ParsedAccountData> }[] | undefined
  >(undefined);
  const [solanaCustodyTokensLoading, setSolanaCustodyTokensLoading] =
    useState(false);
  const [solanaCustodyTokensError, setSolanaCustodyTokensError] = useState("");
  const mintAddresses = useMemo(() => {
    const addresses: string[] = [];
    solanaCustodyTokens?.forEach((item) => {
      const mintKey = item.account.data.parsed?.info?.mint?.toString();
      if (mintKey) {
        addresses.push(mintKey);
      }
    });
    return addresses;
  }, [solanaCustodyTokens]);

  const solanaMetadata = useMetadata(CHAIN_ID_SOLANA, mintAddresses);

  const { isLoading: isTerraLoading, terraTVL } = useTerraTVL();

  const solanaTVL = useMemo(
    () => calcSolanaTVL(solanaCustodyTokens, solanaMetadata),
    [solanaCustodyTokens, solanaMetadata]
  );
  const ethTVL = useMemo(
    () => calcEvmTVL(ethCovalentData, CHAIN_ID_ETH),
    [ethCovalentData]
  );
  const bscTVL = useMemo(
    () => calcEvmTVL(bscCovalentData, CHAIN_ID_BSC),
    [bscCovalentData]
  );

  useEffect(() => {
    let cancelled = false;
    setEthCovalentIsLoading(true);
    axios
      .get(
        COVALENT_GET_TOKENS_URL(CHAIN_ID_ETH, ETH_TOKEN_BRIDGE_ADDRESS, false)
      )
      .then(
        (results) => {
          if (!cancelled) {
            setEthCovalentData(results.data);
            setEthCovalentIsLoading(false);
          }
        },
        (error) => {
          if (!cancelled) {
            setEthCovalentError("Unable to retrieve Ethereum TVL.");
            setEthCovalentIsLoading(false);
          }
        }
      );
  }, []);

  useEffect(() => {
    let cancelled = false;
    setBscCovalentIsLoading(true);
    axios
      .get(
        COVALENT_GET_TOKENS_URL(CHAIN_ID_BSC, BSC_TOKEN_BRIDGE_ADDRESS, false)
      )
      .then(
        (results) => {
          if (!cancelled) {
            setBscCovalentData(results.data);
            setBscCovalentIsLoading(false);
          }
        },
        (error) => {
          if (!cancelled) {
            setBscCovalentError("Unable to retrieve BSC TVL.");
            setBscCovalentIsLoading(false);
          }
        }
      );
  }, []);

  useEffect(() => {
    let cancelled = false;
    const connection = new Connection(SOLANA_HOST, "confirmed");
    connection
      .getParsedTokenAccountsByOwner(new PublicKey(SOL_CUSTODY_ADDRESS), {
        programId: TOKEN_PROGRAM_ID,
      })
      .then(
        (results) => {
          if (!cancelled) {
            setSolanaCustodyTokens(results.value);
            setSolanaCustodyTokensLoading(false);
          }
        },
        (error) => {
          if (!cancelled) {
            setSolanaCustodyTokensLoading(false);
            setSolanaCustodyTokensError(
              "Unable to retrieve Solana locked tokens."
            );
          }
        }
      );
  }, []);

  return useMemo(() => {
    const tvlArray = [...ethTVL, ...bscTVL, ...solanaTVL, ...terraTVL];

    return {
      isFetching:
        ethCovalentIsLoading ||
        bscCovalentIsLoading ||
        solanaCustodyTokensLoading ||
        isTerraLoading,
      error: ethCovalentError || bscCovalentError || solanaCustodyTokensError,
      receivedAt: null,
      data: tvlArray,
    };
  }, [
    ethCovalentError,
    ethCovalentIsLoading,
    bscCovalentError,
    bscCovalentIsLoading,
    ethTVL,
    bscTVL,
    solanaTVL,
    solanaCustodyTokensError,
    solanaCustodyTokensLoading,
    isTerraLoading,
    terraTVL,
  ]);
};

export default useTVL;
