import { CHAIN_ID_ETH, CHAIN_ID_SOLANA } from "@certusone/wormhole-sdk";
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
  COVALENT_GET_TOKENS_URL,
  ETH_TOKEN_BRIDGE_ADDRESS,
  SOLANA_HOST,
  SOL_CUSTODY_ADDRESS,
} from "../utils/consts";
import useMetadata, { GenericMetadata } from "./useMetadata";

export type TVL = {
  logo?: string;
  symbol?: string;
  name?: string;
  amount: string;
  totalValue?: number;
  quotePrice?: number;
  assetAddress: string;
  originChain: string;
};

const calcEthTVL = (covalentReport: any): TVL[] => {
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
        originChain: "Ethereum",
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
      originChain: "Solana",
    });
  });

  return output;
};

const useTVL = (): DataWrapper<TVL[]> => {
  const [covalentData, setCovalentData] = useState(undefined);
  const [covalentIsLoading, setCovalentIsLoading] = useState(false);
  const [covalentError, setCovalentError] = useState("");

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

  const solanaTVL = useMemo(
    () => calcSolanaTVL(solanaCustodyTokens, solanaMetadata),
    [solanaCustodyTokens, solanaMetadata]
  );
  const ethTVL = useMemo(() => calcEthTVL(covalentData), [covalentData]);

  useEffect(() => {
    let cancelled = false;
    setCovalentIsLoading(true);
    axios
      .get(
        COVALENT_GET_TOKENS_URL(CHAIN_ID_ETH, ETH_TOKEN_BRIDGE_ADDRESS, false)
      )
      .then(
        (results) => {
          if (!cancelled) {
            setCovalentData(results.data);
            setCovalentIsLoading(false);
          }
        },
        (error) => {
          if (!cancelled) {
            setCovalentError("Unable to retrieve Ethereum TVL.");
            setCovalentIsLoading(false);
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
    const tvlArray = [...ethTVL, ...solanaTVL];

    return {
      isFetching: covalentIsLoading || solanaCustodyTokensLoading,
      error: covalentError || solanaCustodyTokensError,
      receivedAt: null,
      data: tvlArray,
    };
  }, [
    covalentError,
    covalentIsLoading,
    ethTVL,
    solanaTVL,
    solanaCustodyTokensError,
    solanaCustodyTokensLoading,
  ]);
};

export default useTVL;
