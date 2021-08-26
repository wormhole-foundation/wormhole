import {
  CHAIN_ID_ETH,
  CHAIN_ID_SOLANA,
  CHAIN_ID_TERRA,
  TokenImplementation__factory,
} from "@certusone/wormhole-sdk";
import { Connection, PublicKey } from "@solana/web3.js";
import { useConnectedWallet } from "@terra-money/wallet-provider";
import { formatUnits } from "ethers/lib/utils";
import { useEffect } from "react";
import { useDispatch, useSelector } from "react-redux";
import { useEthereumProvider } from "../contexts/EthereumProviderContext";
import { useSolanaWallet } from "../contexts/SolanaWalletContext";
import {
  selectTransferSourceAsset,
  selectTransferSourceChain,
  selectTransferTargetAsset,
  selectTransferTargetChain,
} from "../store/selectors";
import {
  setSourceParsedTokenAccount,
  setTargetParsedTokenAccount,
} from "../store/transferSlice";
import { SOLANA_HOST } from "../utils/consts";

function createParsedTokenAccount(
  publicKey: PublicKey | undefined,
  amount: string,
  decimals: number,
  uiAmount: number,
  uiAmountString: string
) {
  return {
    publicKey: publicKey?.toString(),
    amount,
    decimals,
    uiAmount,
    uiAmountString,
  };
}

/**
 * Fetches the balance of an asset for the connected wallet
 * @param sourceOrTarget determines whether this will fetch balance for the source or target account. Not intended to be switched on the same hook!
 */
function useGetBalanceEffect(sourceOrTarget: "source" | "target") {
  const dispatch = useDispatch();
  const setAction =
    sourceOrTarget === "source"
      ? setSourceParsedTokenAccount
      : setTargetParsedTokenAccount;
  const lookupChain = useSelector(
    sourceOrTarget === "source"
      ? selectTransferSourceChain
      : selectTransferTargetChain
  );
  const lookupAsset = useSelector(
    sourceOrTarget === "source"
      ? selectTransferSourceAsset
      : selectTransferTargetAsset
  );
  const solanaWallet = useSolanaWallet();
  const solPK = solanaWallet?.publicKey;
  const terraWallet = useConnectedWallet();
  const { provider, signerAddress } = useEthereumProvider();
  useEffect(() => {
    // TODO: loading state
    dispatch(setAction(undefined));
    if (!lookupAsset) {
      return;
    }
    let cancelled = false;

    if (lookupChain === CHAIN_ID_TERRA && terraWallet) {
      dispatch(
        setSourceParsedTokenAccount(
          createParsedTokenAccount(undefined, "0", 0, 0, "0")
        )
      );
    }
    if (lookupChain === CHAIN_ID_SOLANA && solPK) {
      let mint;
      try {
        mint = new PublicKey(lookupAsset);
      } catch (e) {
        return;
      }
      const connection = new Connection(SOLANA_HOST, "finalized");
      connection
        .getParsedTokenAccountsByOwner(solPK, { mint })
        .then(({ value }) => {
          if (!cancelled) {
            if (value.length) {
              // TODO: allow selection between these target accounts
              dispatch(
                setAction(
                  createParsedTokenAccount(
                    value[0].pubkey,
                    value[0].account.data.parsed?.info?.tokenAmount?.amount,
                    value[0].account.data.parsed?.info?.tokenAmount?.decimals,
                    value[0].account.data.parsed?.info?.tokenAmount?.uiAmount,
                    value[0].account.data.parsed?.info?.tokenAmount
                      ?.uiAmountString
                  )
                )
              );
            } else {
              // TODO: error state
            }
          }
        })
        .catch(() => {
          if (!cancelled) {
            // TODO: error state
          }
        });
    }
    if (lookupChain === CHAIN_ID_ETH && provider && signerAddress) {
      const token = TokenImplementation__factory.connect(lookupAsset, provider);
      token
        .decimals()
        .then((decimals) => {
          token.balanceOf(signerAddress).then((n) => {
            if (!cancelled) {
              dispatch(
                setAction(
                  // TODO: verify accuracy
                  createParsedTokenAccount(
                    undefined,
                    n.toString(),
                    decimals,
                    Number(formatUnits(n, decimals)),
                    formatUnits(n, decimals)
                  )
                )
              );
            }
          });
        })
        .catch(() => {
          if (!cancelled) {
            // TODO: error state
          }
        });
    }
    return () => {
      cancelled = true;
    };
  }, [
    dispatch,
    solanaWallet,
    terraWallet,
    sourceOrTarget,
    setAction,
    lookupChain,
    lookupAsset,
    solPK,
    provider,
    signerAddress,
  ]);
}

export default useGetBalanceEffect;
