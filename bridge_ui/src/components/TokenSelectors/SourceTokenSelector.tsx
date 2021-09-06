//import Autocomplete from '@material-ui/lab/Autocomplete';
import {
  CHAIN_ID_ETH,
  CHAIN_ID_SOLANA,
  CHAIN_ID_TERRA,
} from "@certusone/wormhole-sdk";
import { TextField, Typography } from "@material-ui/core";
import { useCallback } from "react";
import { useDispatch, useSelector } from "react-redux";
import useGetSourceParsedTokens from "../../hooks/useGetSourceParsedTokenAccounts";
import useIsWalletReady from "../../hooks/useIsWalletReady";
import {
  selectTransferSourceChain,
  selectTransferSourceParsedTokenAccount,
} from "../../store/selectors";
import {
  ParsedTokenAccount,
  setSourceParsedTokenAccount,
  setSourceWalletAddress,
} from "../../store/transferSlice";
import EthereumSourceTokenSelector from "./EthereumSourceTokenSelector";
import SolanaSourceTokenSelector from "./SolanaSourceTokenSelector";
import TerraSourceTokenSelector from "./TerraSourceTokenSelector";

type TokenSelectorProps = {
  disabled: boolean;
};

export const TokenSelector = (props: TokenSelectorProps) => {
  const { disabled } = props;
  const dispatch = useDispatch();

  const lookupChain = useSelector(selectTransferSourceChain);
  const sourceParsedTokenAccount = useSelector(
    selectTransferSourceParsedTokenAccount
  );
  const walletIsReady = useIsWalletReady(lookupChain);

  const handleOnChange = useCallback(
    (newTokenAccount: ParsedTokenAccount | null) => {
      if (!newTokenAccount) {
        dispatch(setSourceParsedTokenAccount(undefined));
        dispatch(setSourceWalletAddress(undefined));
      } else if (newTokenAccount !== undefined && walletIsReady.walletAddress) {
        dispatch(setSourceParsedTokenAccount(newTokenAccount));
        dispatch(setSourceWalletAddress(walletIsReady.walletAddress));
      }
    },
    [dispatch, walletIsReady]
  );

  const maps = useGetSourceParsedTokens();

  //This is only for errors so bad that we shouldn't even mount the component
  const fatalError =
    lookupChain !== CHAIN_ID_ETH &&
    lookupChain !== CHAIN_ID_TERRA &&
    maps?.tokenAccounts?.error; //Terra & ETH can proceed because it has advanced mode

  const content = fatalError ? (
    <Typography>{fatalError}</Typography>
  ) : lookupChain === CHAIN_ID_SOLANA ? (
    <SolanaSourceTokenSelector
      value={sourceParsedTokenAccount || null}
      onChange={handleOnChange}
      disabled={disabled}
      accounts={maps?.tokenAccounts?.data || []}
      solanaTokenMap={maps?.tokenMap}
      metaplexData={maps?.metaplex}
      mintAccounts={maps?.mintAccounts}
    />
  ) : lookupChain === CHAIN_ID_ETH ? (
    <EthereumSourceTokenSelector
      value={sourceParsedTokenAccount || null}
      disabled={disabled}
      onChange={handleOnChange}
      covalent={maps?.covalent || undefined}
      tokenAccounts={maps?.tokenAccounts} //TODO standardize
    />
  ) : lookupChain === CHAIN_ID_TERRA ? (
    <TerraSourceTokenSelector
      value={sourceParsedTokenAccount || null}
      disabled={disabled}
      onChange={handleOnChange}
      tokenMap={maps?.terraTokenMap}
    />
  ) : (
    <TextField
      placeholder="Asset"
      fullWidth
      value={"Not Implemented"}
      disabled={true}
    />
  );

  return <div>{content}</div>;
};
