import {
  Bridge__factory,
  CHAIN_ID_ETH,
  CHAIN_ID_SOLANA,
  CHAIN_ID_TERRA,
} from "@certusone/wormhole-sdk";
import { ethers } from "@certusone/wormhole-sdk/node_modules/ethers";
import { Dispatch } from "@reduxjs/toolkit";
import { TOKEN_PROGRAM_ID } from "@solana/spl-token";
import {
  AccountInfo,
  Connection,
  ParsedAccountData,
  PublicKey,
} from "@solana/web3.js";
import axios from "axios";
import { formatUnits } from "ethers/lib/utils";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useDispatch, useSelector } from "react-redux";
import {
  Provider,
  useEthereumProvider,
} from "../contexts/EthereumProviderContext";
import { useSolanaWallet } from "../contexts/SolanaWalletContext";
import {
  errorSourceParsedTokenAccounts as errorSourceParsedTokenAccountsNFT,
  fetchSourceParsedTokenAccounts as fetchSourceParsedTokenAccountsNFT,
  NFTParsedTokenAccount,
  receiveSourceParsedTokenAccounts as receiveSourceParsedTokenAccountsNFT,
  setSourceParsedTokenAccount as setSourceParsedTokenAccountNFT,
  setSourceParsedTokenAccounts as setSourceParsedTokenAccountsNFT,
  setSourceWalletAddress as setSourceWalletAddressNFT,
} from "../store/nftSlice";
import {
  selectNFTSourceChain,
  selectNFTSourceParsedTokenAccounts,
  selectNFTSourceWalletAddress,
  selectSourceWalletAddress,
  selectTransferSourceChain,
  selectTransferSourceParsedTokenAccounts,
} from "../store/selectors";
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
  COVALENT_GET_TOKENS_URL,
  ETH_TOKEN_BRIDGE_ADDRESS,
  SOLANA_HOST,
  WETH_ADDRESS,
  WETH_DECIMALS,
} from "../utils/consts";
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
  uiAmountString: string,
  symbol?: string,
  name?: string,
  logo?: string,
  isNativeAsset?: boolean
): ParsedTokenAccount {
  return {
    publicKey: publicKey,
    mintKey: mintKey,
    amount,
    decimals,
    uiAmount,
    uiAmountString,
    symbol,
    name,
    logo,
    isNativeAsset,
  };
}

export function createNFTParsedTokenAccount(
  publicKey: string,
  mintKey: string,
  amount: string,
  decimals: number,
  uiAmount: number,
  uiAmountString: string,
  tokenId: string,
  symbol?: string,
  uri?: string,
  animation_url?: string,
  external_url?: string,
  image?: string,
  image_256?: string,
  name?: string
): NFTParsedTokenAccount {
  return {
    publicKey,
    mintKey,
    amount,
    decimals,
    uiAmount,
    uiAmountString,
    tokenId,
    uri,
    animation_url,
    external_url,
    image,
    image_256,
    symbol,
    name,
  };
}

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
    symbol: covalent.contract_ticker_symbol,
    name: covalent.contract_name,
    logo: covalent.logo_url,
  };
};

const createNativeEthParsedTokenAccount = (
  provider: Provider,
  signerAddress: string | undefined
) => {
  return !(provider && signerAddress)
    ? Promise.reject()
    : provider.getBalance(signerAddress).then((balanceInWei) => {
        const balanceInEth = ethers.utils.formatEther(balanceInWei);
        return createParsedTokenAccount(
          signerAddress, //public key
          WETH_ADDRESS, //Mint key, On the other side this will be WETH, so this is hopefully a white lie.
          balanceInWei.toString(), //amount, in wei
          WETH_DECIMALS, //Luckily both ETH and WETH have 18 decimals, so this should not be an issue.
          parseFloat(balanceInEth), //This loses precision, but is a limitation of the current datamodel. This field is essentially deprecated
          balanceInEth.toString(), //This is the actual display field, which has full precision.
          "ETH", //A white lie for display purposes
          "Ethereum", //A white lie for display purposes
          undefined, //TODO logo
          true //isNativeAsset
        );
      });
};

const createNFTParsedTokenAccountFromCovalent = (
  walletAddress: string,
  covalent: CovalentData,
  nft_data: CovalentNFTData
): NFTParsedTokenAccount => {
  return {
    publicKey: walletAddress,
    mintKey: covalent.contract_address,
    amount: nft_data.token_balance,
    decimals: covalent.contract_decimals,
    uiAmount: Number(
      formatUnits(nft_data.token_balance, covalent.contract_decimals)
    ),
    uiAmountString: formatUnits(
      nft_data.token_balance,
      covalent.contract_decimals
    ),
    symbol: covalent.contract_ticker_symbol,
    logo: covalent.logo_url,
    tokenId: nft_data.token_id,
    uri: nft_data.token_url,
    animation_url: nft_data.external_data.animation_url,
    external_url: nft_data.external_data.external_url,
    image: nft_data.external_data.image,
    image_256: nft_data.external_data.image_256,
    name: nft_data.external_data.name,
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
  nft_data?: CovalentNFTData[];
};

export type CovalentNFTExternalData = {
  animation_url: string | null;
  external_url: string | null;
  image: string;
  image_256: string;
  name: string;
};

export type CovalentNFTData = {
  token_id: string;
  token_balance: string;
  external_data: CovalentNFTExternalData;
  token_url: string;
};

const getEthereumAccountsCovalent = async (
  walletAddress: string,
  nft?: boolean
): Promise<CovalentData[]> => {
  const url = COVALENT_GET_TOKENS_URL(CHAIN_ID_ETH, walletAddress, nft);

  try {
    const output = [] as CovalentData[];
    const response = await axios.get(url);
    const tokens = response.data.data.items;

    if (tokens instanceof Array && tokens.length) {
      for (const item of tokens) {
        // TODO: filter?
        if (
          item.contract_decimals !== undefined &&
          item.contract_ticker_symbol &&
          item.contract_address &&
          item.balance &&
          item.balance !== "0" &&
          (nft
            ? item.supports_erc?.includes("erc721")
            : item.supports_erc?.includes("erc20"))
        ) {
          output.push({ ...item } as CovalentData);
        }
      }
    }

    return output;
  } catch (error) {
    return Promise.reject("Unable to retrieve your Ethereum Tokens.");
  }
};

const getSolanaParsedTokenAccounts = (
  walletAddress: string,
  dispatch: Dispatch,
  nft: boolean
) => {
  const connection = new Connection(SOLANA_HOST, "finalized");
  dispatch(
    nft ? fetchSourceParsedTokenAccountsNFT() : fetchSourceParsedTokenAccounts()
  );
  return connection
    .getParsedTokenAccountsByOwner(new PublicKey(walletAddress), {
      programId: new PublicKey(TOKEN_PROGRAM_ID),
    })
    .then(
      (result) => {
        const mappedItems = result.value.map((item) =>
          createParsedTokenAccountFromInfo(item.pubkey, item.account)
        );
        dispatch(
          nft
            ? receiveSourceParsedTokenAccountsNFT(mappedItems)
            : receiveSourceParsedTokenAccounts(mappedItems)
        );
      },
      (error) => {
        dispatch(
          nft
            ? errorSourceParsedTokenAccountsNFT("Failed to load NFT metadata")
            : errorSourceParsedTokenAccounts("Failed to load token metadata.")
        );
      }
    );
};

/**
 * Fetches the balance of an asset for the connected wallet
 * This should handle every type of chain in the future, but only reads the Transfer state.
 */
function useGetAvailableTokens(nft: boolean = false) {
  const dispatch = useDispatch();

  const tokenAccounts = useSelector(
    nft
      ? selectNFTSourceParsedTokenAccounts
      : selectTransferSourceParsedTokenAccounts
  );

  const lookupChain = useSelector(
    nft ? selectNFTSourceChain : selectTransferSourceChain
  );
  const solanaWallet = useSolanaWallet();
  const solPK = solanaWallet?.publicKey;
  const { provider, signer, signerAddress } = useEthereumProvider();

  const [covalent, setCovalent] = useState<any>(undefined);
  const [covalentLoading, setCovalentLoading] = useState(false);
  const [covalentError, setCovalentError] = useState<string | undefined>(
    undefined
  );

  const [ethNativeAccount, setEthNativeAccount] = useState<any>(undefined);
  const [ethNativeAccountLoading, setEthNativeAccountLoading] = useState(false);
  const [ethNativeAccountError, setEthNativeAccountError] = useState<
    string | undefined
  >(undefined);

  const [solanaMintAccounts, setSolanaMintAccounts] = useState<any>(undefined);
  const [solanaMintAccountsLoading, setSolanaMintAccountsLoading] =
    useState(false);
  const [solanaMintAccountsError, setSolanaMintAccountsError] = useState<
    string | undefined
  >(undefined);

  const selectedSourceWalletAddress = useSelector(
    nft ? selectNFTSourceWalletAddress : selectSourceWalletAddress
  );
  const currentSourceWalletAddress: string | undefined =
    lookupChain === CHAIN_ID_ETH
      ? signerAddress
      : lookupChain === CHAIN_ID_SOLANA
      ? solPK?.toString()
      : undefined;

  const resetSourceAccounts = useCallback(() => {
    dispatch(
      nft
        ? setSourceWalletAddressNFT(undefined)
        : setSourceWalletAddress(undefined)
    );
    dispatch(
      nft
        ? setSourceParsedTokenAccountNFT(undefined)
        : setSourceParsedTokenAccount(undefined)
    );
    dispatch(
      nft
        ? setSourceParsedTokenAccountsNFT(undefined)
        : setSourceParsedTokenAccounts(undefined)
    );
    !nft && dispatch(setAmount(""));
    setCovalent(undefined); //These need to be included in the reset because they have balances on them.
    setCovalentLoading(false);
    setCovalentError("");

    setEthNativeAccount(undefined);
    setEthNativeAccountLoading(false);
    setEthNativeAccountError("");
  }, [setCovalent, dispatch, nft]);

  //TODO this useEffect could be somewhere else in the codebase
  //It resets the SourceParsedTokens accounts when the wallet changes
  useEffect(() => {
    if (
      selectedSourceWalletAddress !== undefined &&
      currentSourceWalletAddress !== undefined &&
      currentSourceWalletAddress !== selectedSourceWalletAddress
    ) {
      resetSourceAccounts();
      return;
    } else {
    }
  }, [
    selectedSourceWalletAddress,
    currentSourceWalletAddress,
    dispatch,
    resetSourceAccounts,
  ]);

  //Solana accountinfos load
  useEffect(() => {
    if (lookupChain === CHAIN_ID_SOLANA && solPK) {
      if (
        !(tokenAccounts.data || tokenAccounts.isFetching || tokenAccounts.error)
      ) {
        getSolanaParsedTokenAccounts(solPK.toString(), dispatch, nft);
      }
    }

    return () => {};
  }, [dispatch, solanaWallet, lookupChain, solPK, tokenAccounts, nft]);

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
    // mintAddresses.push("4QixXecTZ4zdZGa39KH8gVND5NZ2xcaB12wiBhE4S7rn");
    //SOLT devnet token
    // mintAddresses.push("2WDq7wSs9zYrpx2kbHDA4RUTRch2CCTP6ZWaH4GNfnQQ");

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

  //Ethereum native asset load
  useEffect(() => {
    let cancelled = false;
    if (
      signerAddress &&
      lookupChain === CHAIN_ID_ETH &&
      !ethNativeAccount &&
      !nft
    ) {
      setEthNativeAccountLoading(true);
      createNativeEthParsedTokenAccount(provider, signerAddress).then(
        (result) => {
          console.log("create native account returned with value", result);
          if (!cancelled) {
            setEthNativeAccount(result);
            setEthNativeAccountLoading(false);
            setEthNativeAccountError("");
          }
        },
        (error) => {
          if (!cancelled) {
            setEthNativeAccount(undefined);
            setEthNativeAccountLoading(false);
            setEthNativeAccountError("Unable to retrieve your ETH balance.");
          }
        }
      );
    }

    return () => {
      cancelled = true;
    };
  }, [lookupChain, provider, signerAddress, nft, ethNativeAccount]);

  //Ethereum covalent accounts load
  useEffect(() => {
    //const testWallet = "0xf60c2ea62edbfe808163751dd0d8693dcb30019c";
    // const nftTestWallet1 = "0x3f304c6721f35ff9af00fd32650c8e0a982180ab";
    // const nftTestWallet2 = "0x98ed231428088eb440e8edb5cc8d66dcf913b86e";
    let cancelled = false;
    const walletAddress = signerAddress;
    if (walletAddress && lookupChain === CHAIN_ID_ETH && !covalent) {
      //TODO less cancel
      !cancelled && setCovalentLoading(true);
      !cancelled &&
        dispatch(
          nft
            ? fetchSourceParsedTokenAccountsNFT()
            : fetchSourceParsedTokenAccounts()
        );
      getEthereumAccountsCovalent(walletAddress, nft).then(
        (accounts) => {
          !cancelled && setCovalentLoading(false);
          !cancelled && setCovalentError(undefined);
          !cancelled && setCovalent(accounts);
          !cancelled &&
            dispatch(
              nft
                ? receiveSourceParsedTokenAccountsNFT(
                    accounts.reduce((arr, current) => {
                      if (current.nft_data) {
                        current.nft_data.forEach((x) =>
                          arr.push(
                            createNFTParsedTokenAccountFromCovalent(
                              walletAddress,
                              current,
                              x
                            )
                          )
                        );
                      }
                      return arr;
                    }, [] as NFTParsedTokenAccount[])
                  )
                : receiveSourceParsedTokenAccounts(
                    accounts.map((x) =>
                      createParsedTokenAccountFromCovalent(walletAddress, x)
                    )
                  )
            );
        },
        () => {
          !cancelled &&
            dispatch(
              nft
                ? errorSourceParsedTokenAccountsNFT(
                    "Cannot load your Ethereum NFTs at the moment."
                  )
                : errorSourceParsedTokenAccounts(
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
    }
  }, [lookupChain, provider, signerAddress, dispatch, nft, covalent]);

  //Terra accounts load
  //At present, we don't have any mechanism for doing this.
  useEffect(() => {}, []);

  const ethAccounts = useMemo(() => {
    const output = { ...tokenAccounts };
    output.data = output.data?.slice() || [];
    output.isFetching = output.isFetching || ethNativeAccountLoading;
    output.error = output.error || ethNativeAccountError;
    ethNativeAccount && output.data && output.data.unshift(ethNativeAccount);
    return output;
  }, [
    ethNativeAccount,
    ethNativeAccountLoading,
    ethNativeAccountError,
    tokenAccounts,
  ]);

  return lookupChain === CHAIN_ID_SOLANA
    ? {
        tokenAccounts: tokenAccounts,
        mintAccounts: {
          data: solanaMintAccounts,
          isFetching: solanaMintAccountsLoading,
          error: solanaMintAccountsError,
          receivedAt: null, //TODO
        },
        resetAccounts: resetSourceAccounts,
      }
    : lookupChain === CHAIN_ID_ETH
    ? {
        tokenAccounts: ethAccounts,
        covalent: {
          data: covalent,
          isFetching: covalentLoading,
          error: covalentError,
          receivedAt: null, //TODO
        },
        resetAccounts: resetSourceAccounts,
      }
    : lookupChain === CHAIN_ID_TERRA
    ? {
        resetAccounts: resetSourceAccounts,
      }
    : undefined;
}

export default useGetAvailableTokens;
