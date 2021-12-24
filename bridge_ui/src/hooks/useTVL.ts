import {
  ChainId,
  CHAIN_ID_AVAX,
  CHAIN_ID_BSC,
  CHAIN_ID_ETH,
  CHAIN_ID_POLYGON,
  CHAIN_ID_SOLANA,
  CHAIN_ID_TERRA,
} from "@certusone/wormhole-sdk";
import { formatUnits } from "@ethersproject/units";
import { TOKEN_PROGRAM_ID } from "@solana/spl-token";
import { TokenInfo } from "@solana/spl-token-registry";
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
  AVAX_TOKEN_BRIDGE_ADDRESS,
  BSC_TOKEN_BRIDGE_ADDRESS,
  CHAINS_BY_ID,
  COVALENT_GET_TOKENS_URL,
  ETH_TOKEN_BRIDGE_ADDRESS,
  logoOverrides,
  POLYGON_TOKEN_BRIDGE_ADDRESS,
  SOLANA_HOST,
  SOL_CUSTODY_ADDRESS,
  TERRA_SWAPRATE_URL,
  TERRA_TOKEN_BRIDGE_ADDRESS,
} from "../utils/consts";
import { priceStore, serumMarkets } from "../utils/SolanaPriceStore";
import {
  formatNativeDenom,
  getNativeTerraIcon,
  NATIVE_TERRA_DECIMALS,
} from "../utils/terra";
import useMetadata, { GenericMetadata } from "./useMetadata";
import useSolanaTokenMap from "./useSolanaTokenMap";
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
  decimals?: number;
};

const BAD_PRICES_BY_CHAIN = {
  [CHAIN_ID_BSC]: [
    "0x04132bf45511d03a58afd4f1d36a29d229ccc574",
    "0xa79bd679ce21a2418be9e6f88b2186c9986bbe7d",
    "0x931c3987040c90b6db09981c7c91ba155d3fa31f",
  ],
  [CHAIN_ID_ETH]: ["0x3845badade8e6dff049820680d1f14bd3903a5d0"],
};

const calcEvmTVL = (covalentReport: any, chainId: ChainId): TVL[] => {
  const output: TVL[] = [];
  if (!covalentReport?.data?.items?.length) {
    return [];
  }

  covalentReport.data.items.forEach((item: any) => {
    if (item.balance > 0 && item.contract_address) {
      const hasUnreliablePrice =
        BAD_PRICES_BY_CHAIN[chainId]?.includes(item.contract_address) ||
        item.quote_rate > 1000000;
      output.push({
        logo:
          logoOverrides.get(item.contract_address) ||
          item.logo_url ||
          undefined,
        symbol: item.contract_ticker_symbol || undefined,
        name: item.contract_name || undefined,
        amount: formatUnits(item.balance, item.contract_decimals),
        totalValue: hasUnreliablePrice ? 0 : item.quote,
        quotePrice: hasUnreliablePrice ? 0 : item.quote_rate,
        assetAddress: item.contract_address,
        originChainId: chainId,
        originChain: CHAINS_BY_ID[chainId].name,
        decimals: item.contract_decimals,
      });
    }
  });

  return output;
};
const calcSolanaTVL = (
  accounts:
    | { pubkey: PublicKey; account: AccountInfo<ParsedAccountData> }[]
    | undefined,
  metaData: DataWrapper<Map<string, GenericMetadata>>,
  solanaPrices: DataWrapper<Map<string, number | undefined>>
) => {
  const output: TVL[] = [];
  if (
    !accounts ||
    !accounts.length ||
    metaData.isFetching ||
    metaData.error ||
    !metaData.data ||
    solanaPrices.isFetching ||
    !solanaPrices.data
  ) {
    return output;
  }

  accounts.forEach((item) => {
    const genericMetadata = metaData.data?.get(
      item.account.data.parsed?.info?.mint?.toString()
    );
    const mint = item.account.data.parsed?.info?.mint?.toString();
    const price = solanaPrices?.data?.get(mint);
    output.push({
      logo: genericMetadata?.logo || undefined,
      symbol: genericMetadata?.symbol || undefined,
      name: genericMetadata?.tokenName || undefined,
      amount: item.account.data.parsed?.info?.tokenAmount?.uiAmount || "0", //Should always be defined.
      totalValue: price
        ? parseFloat(
            item.account.data.parsed?.info?.tokenAmount?.uiAmount || "0"
          ) * price
        : undefined,
      quotePrice: price,
      assetAddress: mint,
      originChainId: CHAIN_ID_SOLANA,
      originChain: "Solana",
      decimals: item.account.data.parsed?.info?.tokenAmount?.decimals,
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
          decimals: NATIVE_TERRA_DECIMALS,
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

const useSolanaPrices = (
  mintAddresses: string[],
  tokenMap: DataWrapper<TokenInfo[]>
) => {
  const [isLoading, setIsLoading] = useState(false);
  const [priceMap, setPriceMap] = useState<Map<
    string,
    number | undefined
  > | null>(null);
  const [error] = useState("");

  useEffect(() => {
    let cancelled = false;

    if (!mintAddresses || !mintAddresses.length || !tokenMap.data) {
      return;
    }

    const relevantMarkets: {
      publicKey?: PublicKey;
      name: string;
      deprecated?: boolean;
      mintAddress: string;
    }[] = [];
    mintAddresses.forEach((address) => {
      const tokenInfo = tokenMap.data?.find((x) => x.address === address);
      const relevantMarket = tokenInfo && serumMarkets[tokenInfo.symbol];
      if (relevantMarket) {
        relevantMarkets.push({ ...relevantMarket, mintAddress: address });
      }
    });

    setIsLoading(true);
    const priceMap: Map<string, number | undefined> = new Map();
    const connection = new Connection(SOLANA_HOST);
    const promises: Promise<void>[] = [];
    //Load all the revelevant markets into the priceMap
    relevantMarkets.forEach((market) => {
      const marketName: string = market.name;
      promises.push(
        priceStore
          .getPrice(connection, marketName)
          .then((result) => {
            priceMap.set(market.mintAddress, result);
          })
          .catch((e) => {
            //Do nothing, we just won't load this price.
            return Promise.resolve();
          })
      );
    });

    Promise.all(promises).then(() => {
      //By this point all the relevant markets are loaded.
      if (!cancelled) {
        setPriceMap(priceMap);
        setIsLoading(false);
      }
    });

    return () => {
      cancelled = true;
      return;
    };
  }, [mintAddresses, tokenMap.data]);

  return useMemo(() => {
    return {
      isFetching: isLoading,
      data: priceMap || null,
      error: error,
      receivedAt: null,
    };
  }, [error, priceMap, isLoading]);
};

const useTVL = (): DataWrapper<TVL[]> => {
  const [ethCovalentData, setEthCovalentData] = useState(undefined);
  const [ethCovalentIsLoading, setEthCovalentIsLoading] = useState(false);
  const [ethCovalentError, setEthCovalentError] = useState("");

  const [bscCovalentData, setBscCovalentData] = useState(undefined);
  const [bscCovalentIsLoading, setBscCovalentIsLoading] = useState(false);
  const [bscCovalentError, setBscCovalentError] = useState("");

  const [polygonCovalentData, setPolygonCovalentData] = useState(undefined);
  const [polygonCovalentIsLoading, setPolygonCovalentIsLoading] =
    useState(false);
  const [polygonCovalentError, setPolygonCovalentError] = useState("");

  const [avaxCovalentData, setAvaxCovalentData] = useState(undefined);
  const [avaxCovalentIsLoading, setAvaxCovalentIsLoading] = useState(false);
  const [avaxCovalentError, setAvaxCovalentError] = useState("");

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
  const solanaTokenMap = useSolanaTokenMap();
  const solanaPrices = useSolanaPrices(mintAddresses, solanaTokenMap);

  const { isLoading: isTerraLoading, terraTVL } = useTerraTVL();

  const solanaTVL = useMemo(
    () => calcSolanaTVL(solanaCustodyTokens, solanaMetadata, solanaPrices),
    [solanaCustodyTokens, solanaMetadata, solanaPrices]
  );
  const ethTVL = useMemo(
    () => calcEvmTVL(ethCovalentData, CHAIN_ID_ETH),
    [ethCovalentData]
  );
  const bscTVL = useMemo(
    () => calcEvmTVL(bscCovalentData, CHAIN_ID_BSC),
    [bscCovalentData]
  );
  const polygonTVL = useMemo(
    () => calcEvmTVL(polygonCovalentData, CHAIN_ID_POLYGON),
    [polygonCovalentData]
  );
  const avaxTVL = useMemo(
    () => calcEvmTVL(avaxCovalentData, CHAIN_ID_AVAX),
    [avaxCovalentData]
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
    setPolygonCovalentIsLoading(true);
    axios
      .get(
        COVALENT_GET_TOKENS_URL(
          CHAIN_ID_POLYGON,
          POLYGON_TOKEN_BRIDGE_ADDRESS,
          false
        )
      )
      .then(
        (results) => {
          if (!cancelled) {
            setPolygonCovalentData(results.data);
            setPolygonCovalentIsLoading(false);
          }
        },
        (error) => {
          if (!cancelled) {
            setPolygonCovalentError("Unable to retrieve Polygon TVL.");
            setPolygonCovalentIsLoading(false);
          }
        }
      );
  }, []);

  useEffect(() => {
    let cancelled = false;
    setAvaxCovalentIsLoading(true);
    axios
      .get(
        COVALENT_GET_TOKENS_URL(CHAIN_ID_AVAX, AVAX_TOKEN_BRIDGE_ADDRESS, false)
      )
      .then(
        (results) => {
          if (!cancelled) {
            setAvaxCovalentData(results.data);
            setAvaxCovalentIsLoading(false);
          }
        },
        (error) => {
          if (!cancelled) {
            setAvaxCovalentError("Unable to retrieve Avax TVL.");
            setAvaxCovalentIsLoading(false);
          }
        }
      );
  }, []);

  useEffect(() => {
    let cancelled = false;
    const connection = new Connection(SOLANA_HOST, "confirmed");
    setSolanaCustodyTokensLoading(true);
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
    const tvlArray = [
      ...ethTVL,
      ...bscTVL,
      ...polygonTVL,
      ...avaxTVL,
      ...solanaTVL,
      ...terraTVL,
    ];

    return {
      isFetching:
        ethCovalentIsLoading ||
        bscCovalentIsLoading ||
        polygonCovalentIsLoading ||
        avaxCovalentIsLoading ||
        solanaCustodyTokensLoading ||
        isTerraLoading,
      error:
        ethCovalentError ||
        bscCovalentError ||
        polygonCovalentError ||
        avaxCovalentError ||
        solanaCustodyTokensError,
      receivedAt: null,
      data: tvlArray,
    };
  }, [
    ethCovalentError,
    ethCovalentIsLoading,
    bscCovalentError,
    bscCovalentIsLoading,
    polygonCovalentError,
    polygonCovalentIsLoading,
    polygonTVL,
    avaxCovalentError,
    avaxCovalentIsLoading,
    avaxTVL,
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
