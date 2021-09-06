import {
  CHAIN_ID_ETH,
  CHAIN_ID_SOLANA,
  CHAIN_ID_TERRA,
} from "@certusone/wormhole-sdk";
import { Dispatch } from "@reduxjs/toolkit";
import { TOKEN_PROGRAM_ID } from "@solana/spl-token";
import { ENV, TokenListProvider } from "@solana/spl-token-registry";
import {
  AccountInfo,
  Connection,
  ParsedAccountData,
  PublicKey,
} from "@solana/web3.js";
import axios from "axios";
import { formatUnits } from "ethers/lib/utils";
import { useEffect, useState } from "react";
import { useDispatch, useSelector } from "react-redux";
import { useEthereumProvider } from "../contexts/EthereumProviderContext";
import { useSolanaWallet } from "../contexts/SolanaWalletContext";
import { DataWrapper } from "../store/helpers";
import {
  selectSolanaTokenMap,
  selectSourceWalletAddress,
  selectTerraTokenMap,
  selectTransferSourceChain,
  selectTransferSourceParsedTokenAccounts,
} from "../store/selectors";
import {
  errorSolanaTokenMap,
  errorTerraTokenMap,
  fetchSolanaTokenMap,
  fetchTerraTokenMap,
  receiveSolanaTokenMap,
  receiveTerraTokenMap,
} from "../store/tokenSlice";
import {
  errorSourceParsedTokenAccounts,
  fetchSourceParsedTokenAccounts,
  ParsedTokenAccount,
  receiveSourceParsedTokenAccounts,
  setAmount,
  setSourceParsedTokenAccount,
  setSourceParsedTokenAccounts,
  setSourceWalletAddress,
} from "../store/transferSlice";
import {
  CLUSTER,
  COVALENT_GET_TOKENS_URL,
  SOLANA_HOST,
  TERRA_TOKEN_METADATA_URL,
} from "../utils/consts";
import {
  decodeMetadata,
  getMetadataAddress,
  Metadata,
} from "../utils/metaplex";
import {
  extractMintAuthorityInfo,
  getMultipleAccountsRPC,
} from "../utils/solana";

export function createParsedTokenAccount(
  publicKey: string,
  mintKey: string,
  amount: string,
  decimals: number,
  uiAmount: number,
  uiAmountString: string
): ParsedTokenAccount {
  return {
    publicKey: publicKey,
    mintKey: mintKey,
    amount,
    decimals,
    uiAmount,
    uiAmountString,
  };
}

export type TerraTokenMetadata = {
  protocol: string;
  symbol: string;
  token: string;
  icon: string;
};

export type TerraTokenMap = {
  mainnet: {
    [address: string]: TerraTokenMetadata;
  };
};

const createParsedTokenAccountFromInfo = (
  pubkey: PublicKey,
  item: AccountInfo<ParsedAccountData>
): ParsedTokenAccount => {
  return {
    publicKey: pubkey?.toString(),
    mintKey: item.data.parsed?.info?.mint?.toString(),
    amount: item.data.parsed?.info?.tokenAmount?.amount,
    decimals: item.data.parsed?.info?.tokenAmount?.decimals,
    uiAmount: item.data.parsed?.info?.tokenAmount?.uiAmount,
    uiAmountString: item.data.parsed?.info?.tokenAmount?.uiAmountString,
  };
};

const createParsedTokenAccountFromCovalent = (
  walletAddress: string,
  covalent: CovalentData
): ParsedTokenAccount => {
  return {
    publicKey: walletAddress,
    mintKey: covalent.contract_address,
    amount: covalent.balance,
    decimals: covalent.contract_decimals,
    uiAmount: Number(formatUnits(covalent.balance, covalent.contract_decimals)),
    uiAmountString: formatUnits(covalent.balance, covalent.contract_decimals),
  };
};

export type CovalentData = {
  contract_decimals: number;
  contract_ticker_symbol: string;
  contract_name: string;
  contract_address: string;
  logo_url: string | undefined;
  balance: string;
  quote: number | undefined;
  quote_rate: number | undefined;
};

const getEthereumAccountsCovalent = async (
  walletAddress: string
): Promise<CovalentData[]> => {
  const url = COVALENT_GET_TOKENS_URL(CHAIN_ID_ETH, walletAddress);

  try {
    const output = [] as CovalentData[];
    const response = await axios.get(url);
    const tokens = response.data.data.items;

    if (tokens instanceof Array && tokens.length) {
      for (const item of tokens) {
        // TODO: filter?
        if (
          item.contract_decimals &&
          item.contract_ticker_symbol &&
          item.contract_address &&
          item.balance &&
          item.supports_erc?.includes("erc20")
        ) {
          output.push({ ...item } as CovalentData);
        }
      }
    }

    return output;
  } catch (error) {
    console.error(error);
    return Promise.reject("Unable to retrieve your Ethereum Tokens.");
  }
};

const environment = CLUSTER === "testnet" ? ENV.Testnet : ENV.MainnetBeta;

const getMetaplexData = async (mintAddresses: string[]) => {
  const promises = [];
  for (const address of mintAddresses) {
    promises.push(getMetadataAddress(address));
  }
  const metaAddresses = await Promise.all(promises);
  const connection = new Connection(SOLANA_HOST, "finalized");
  const results = await getMultipleAccountsRPC(
    connection,
    metaAddresses.map((pair) => pair && pair[0])
  );

  const output = results.map((account) => {
    if (account === null) {
      return undefined;
    } else {
      if (account.data) {
        try {
          const MetadataParsed = decodeMetadata(account.data);
          return MetadataParsed;
        } catch (e) {
          console.error(e);
          return undefined;
        }
      } else {
        return undefined;
      }
    }
  });

  return output;
};

const getSolanaParsedTokenAccounts = (
  walletAddress: string,
  dispatch: Dispatch
) => {
  const connection = new Connection(SOLANA_HOST, "finalized");
  dispatch(fetchSourceParsedTokenAccounts());
  return connection
    .getParsedTokenAccountsByOwner(new PublicKey(walletAddress), {
      programId: new PublicKey(TOKEN_PROGRAM_ID),
    })
    .then(
      (result) => {
        const mappedItems = result.value.map((item) =>
          createParsedTokenAccountFromInfo(item.pubkey, item.account)
        );
        dispatch(receiveSourceParsedTokenAccounts(mappedItems));
      },
      (error) => {
        dispatch(
          errorSourceParsedTokenAccounts("Failed to load token metadata.")
        );
      }
    );
};

const getSolanaTokenMap = (dispatch: Dispatch) => {
  dispatch(fetchSolanaTokenMap());

  new TokenListProvider().resolve().then(
    (tokens) => {
      const tokenList = tokens.filterByChainId(environment).getList();
      dispatch(receiveSolanaTokenMap(tokenList));
    },
    (error) => {
      console.error(error);
      dispatch(errorSolanaTokenMap("Failed to retrieve the Solana token map."));
    }
  );
};
/**
 * Fetches the balance of an asset for the connected wallet
 * This should handle every type of chain in the future, but only reads the Transfer state.
 */
function useGetAvailableTokens() {
  const dispatch = useDispatch();

  const tokenAccounts = useSelector(selectTransferSourceParsedTokenAccounts);
  const solanaTokenMap = useSelector(selectSolanaTokenMap);
  const terraTokenMap = useSelector(selectTerraTokenMap);

  const lookupChain = useSelector(selectTransferSourceChain);
  const solanaWallet = useSolanaWallet();
  const solPK = solanaWallet?.publicKey;
  //const terraWallet = useConnectedWallet(); //TODO
  const { provider, signerAddress } = useEthereumProvider();

  const [metaplex, setMetaplex] = useState<any>(undefined);
  const [metaplexLoading, setMetaplexLoading] = useState(false);
  const [metaplexError, setMetaplexError] = useState(null);

  const [covalent, setCovalent] = useState<any>(undefined);
  const [covalentLoading, setCovalentLoading] = useState(false);
  const [covalentError, setCovalentError] = useState<string | undefined>(
    undefined
  );

  const [solanaMintAccounts, setSolanaMintAccounts] = useState<any>(undefined);
  const [solanaMintAccountsLoading, setSolanaMintAccountsLoading] =
    useState(false);
  const [solanaMintAccountsError, setSolanaMintAccountsError] = useState<
    string | undefined
  >(undefined);

  const selectedSourceWalletAddress = useSelector(selectSourceWalletAddress);
  const currentSourceWalletAddress: string | undefined =
    lookupChain === CHAIN_ID_ETH
      ? signerAddress
      : lookupChain === CHAIN_ID_SOLANA
      ? solPK?.toString()
      : undefined;

  //TODO this useEffect could be somewhere else in the codebase
  //It resets the SourceParsedTokens accounts when the wallet changes
  useEffect(() => {
    if (
      selectedSourceWalletAddress !== undefined &&
      currentSourceWalletAddress !== undefined &&
      currentSourceWalletAddress !== selectedSourceWalletAddress
    ) {
      dispatch(setSourceWalletAddress(undefined));
      dispatch(setSourceParsedTokenAccount(undefined));
      dispatch(setSourceParsedTokenAccounts(undefined));
      dispatch(setAmount(""));
      return;
    } else {
    }
  }, [selectedSourceWalletAddress, currentSourceWalletAddress, dispatch]);

  // Solana metaplex load
  useEffect(() => {
    let cancelled = false;
    if (tokenAccounts.data && lookupChain === CHAIN_ID_SOLANA) {
      setMetaplexLoading(true);
      const accounts = tokenAccounts.data.map((account) => account.mintKey);
      accounts.filter((x) => !!x);
      getMetaplexData(accounts as string[]).then(
        (results) => {
          if (!cancelled) {
            setMetaplex(results);
            setMetaplexLoading(false);
          } else {
          }
        },
        (error) => {
          if (!cancelled) {
            console.error(error);
            setMetaplexLoading(false);
            setMetaplexError(error);
          } else {
          }
        }
      );
    } else {
    }

    return () => {
      cancelled = true;
    };
  }, [tokenAccounts, lookupChain]);

  //Solana token map & accountinfos load
  useEffect(() => {
    if (lookupChain === CHAIN_ID_SOLANA && solPK) {
      if (
        !(tokenAccounts.data || tokenAccounts.isFetching || tokenAccounts.error)
      ) {
        getSolanaParsedTokenAccounts(solPK.toString(), dispatch);
      }
      if (
        !(
          solanaTokenMap.data ||
          solanaTokenMap.isFetching ||
          solanaTokenMap.error
        )
      ) {
        getSolanaTokenMap(dispatch);
      }
    }

    return () => {};
  }, [
    dispatch,
    solanaWallet,
    lookupChain,
    solPK,
    tokenAccounts,
    solanaTokenMap,
  ]);

  //Solana Mint Accounts lookup
  useEffect(() => {
    if (lookupChain !== CHAIN_ID_SOLANA || !tokenAccounts.data?.length) {
      return () => {};
    }

    let cancelled = false;
    setSolanaMintAccountsLoading(true);
    setSolanaMintAccountsError(undefined);
    const mintAddresses = tokenAccounts.data.map((x) => x.mintKey);
    //This is a known wormhole v1 token on testnet
    //mintAddresses.push("4QixXecTZ4zdZGa39KH8gVND5NZ2xcaB12wiBhE4S7rn");

    const connection = new Connection(SOLANA_HOST, "finalized");
    getMultipleAccountsRPC(
      connection,
      mintAddresses.map((x) => new PublicKey(x))
    ).then(
      (results) => {
        if (!cancelled) {
          const output = new Map<String, string | null>();

          results.forEach((result, index) =>
            output.set(
              mintAddresses[index],
              (result && extractMintAuthorityInfo(result)) || null
            )
          );

          setSolanaMintAccounts(output);
          setSolanaMintAccountsLoading(false);
        }
      },
      (error) => {
        if (!cancelled) {
          setSolanaMintAccounts(undefined);
          setSolanaMintAccountsLoading(false);
          setSolanaMintAccountsError(
            "Could not retrieve Solana mint accounts."
          );
        }
      }
    );

    return () => (cancelled = true);
  }, [tokenAccounts.data, lookupChain]);

  //Ethereum accounts load
  useEffect(() => {
    //const testWallet = "0xf60c2ea62edbfe808163751dd0d8693dcb30019c";
    let cancelled = false;
    const walletAddress = signerAddress;
    if (!walletAddress || lookupChain !== CHAIN_ID_ETH) {
      return;
    }
    //TODO less cancel
    !cancelled && setCovalentLoading(true);
    !cancelled && dispatch(fetchSourceParsedTokenAccounts());
    getEthereumAccountsCovalent(walletAddress).then(
      (accounts) => {
        !cancelled && setCovalentLoading(false);
        !cancelled && setCovalentError(undefined);
        !cancelled && setCovalent(accounts);
        !cancelled &&
          dispatch(
            receiveSourceParsedTokenAccounts(
              accounts.map((x) =>
                createParsedTokenAccountFromCovalent(walletAddress, x)
              )
            )
          );
      },
      () => {
        !cancelled &&
          dispatch(
            errorSourceParsedTokenAccounts(
              "Cannot load your Ethereum tokens at the moment."
            )
          );
        !cancelled &&
          setCovalentError("Cannot load your Ethereum tokens at the moment.");
        !cancelled && setCovalentLoading(false);
      }
    );

    return () => {
      cancelled = true;
    };
  }, [lookupChain, provider, signerAddress, dispatch]);

  //Terra accounts load
  //At present, we don't have any mechanism for doing this.
  useEffect(() => {}, []);

  //Terra metadata load
  useEffect(() => {
    let cancelled = false;

    if (terraTokenMap.data || lookupChain !== CHAIN_ID_TERRA) {
      return; //So we don't fetch the whole list on every mount.
    }

    dispatch(fetchTerraTokenMap());
    axios.get(TERRA_TOKEN_METADATA_URL).then(
      (response) => {
        if (!cancelled) {
          //TODO parse this in a safer manner
          dispatch(receiveTerraTokenMap(response.data as TerraTokenMap));
        }
      },
      (error) => {
        if (!cancelled) {
          dispatch(
            errorTerraTokenMap("Failed to retrieve the Terra Token List.")
          );
        }
      }
    );

    return () => {
      cancelled = true;
    };
  }, [lookupChain, terraTokenMap.data, dispatch]);

  return lookupChain === CHAIN_ID_SOLANA
    ? {
        tokenMap: solanaTokenMap,
        tokenAccounts: tokenAccounts,
        metaplex: {
          data: metaplex,
          isFetching: metaplexLoading,
          error: metaplexError,
          receivedAt: null, //TODO
        } as DataWrapper<Metadata[]>,
        mintAccounts: {
          data: solanaMintAccounts,
          isFetching: solanaMintAccountsLoading,
          error: solanaMintAccountsError,
          receivedAt: null, //TODO
        },
      }
    : lookupChain === CHAIN_ID_ETH
    ? {
        tokenAccounts: tokenAccounts,
        covalent: {
          data: covalent,
          isFetching: covalentLoading,
          error: covalentError,
          receivedAt: null, //TODO
        },
      }
    : lookupChain === CHAIN_ID_TERRA
    ? {
        terraTokenMap: terraTokenMap,
      }
    : undefined;
}

export default useGetAvailableTokens;
