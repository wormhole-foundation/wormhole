import { Connection, PublicKey } from "@solana/web3.js";
import { formatUnits } from "ethers/lib/utils";
import { useEffect } from "react";
import { useDispatch, useSelector } from "react-redux";
import { useEthereumProvider } from "../contexts/EthereumProviderContext";
import { useSolanaWallet } from "../contexts/SolanaWalletContext";
import { TokenImplementation__factory } from "../ethers-contracts";
import { selectSourceAsset, selectSourceChain } from "../store/selectors";
import { setSourceParsedTokenAccount } from "../store/transferSlice";
import { CHAIN_ID_ETH, CHAIN_ID_SOLANA, SOLANA_HOST } from "../utils/consts";

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

function useGetBalanceEffect() {
  const dispatch = useDispatch();
  const sourceChain = useSelector(selectSourceChain);
  const sourceAsset = useSelector(selectSourceAsset);
  const { wallet } = useSolanaWallet();
  const solPK = wallet?.publicKey;
  const { provider, signerAddress } = useEthereumProvider();
  useEffect(() => {
    // TODO: loading state
    dispatch(setSourceParsedTokenAccount(undefined));
    if (!sourceAsset) {
      return;
    }
    let cancelled = false;
    if (sourceChain === CHAIN_ID_SOLANA && solPK) {
      let mint;
      try {
        mint = new PublicKey(sourceAsset);
      } catch (e) {
        return;
      }
      const connection = new Connection(SOLANA_HOST, "finalized");
      connection
        .getParsedTokenAccountsByOwner(solPK, { mint })
        .then(({ value }) => {
          if (!cancelled) {
            if (value.length) {
              dispatch(
                setSourceParsedTokenAccount(
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
    if (sourceChain === CHAIN_ID_ETH && provider && signerAddress) {
      const token = TokenImplementation__factory.connect(sourceAsset, provider);
      token
        .decimals()
        .then((decimals) => {
          token.balanceOf(signerAddress).then((n) => {
            if (!cancelled) {
              dispatch(
                setSourceParsedTokenAccount(
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
  }, [dispatch, sourceChain, sourceAsset, solPK, provider, signerAddress]);
}

export default useGetBalanceEffect;
