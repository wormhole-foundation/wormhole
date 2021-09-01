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
import {
  selectTransferSourceChain,
  selectTransferSourceParsedTokenAccount,
} from "../../store/selectors";
import {
  ParsedTokenAccount,
  setSourceParsedTokenAccount,
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
  const handleSolanaOnChange = useCallback(
    (newTokenAccount: ParsedTokenAccount | null) => {
      if (newTokenAccount !== undefined) {
        dispatch(setSourceParsedTokenAccount(newTokenAccount || undefined));
      }
    },
    [dispatch]
  );

  const maps = useGetSourceParsedTokens();

  //This is only for errors so bad that we shouldn't even mount the component
  const fatalError =
    maps?.tokenAccounts?.error &&
    !(lookupChain === CHAIN_ID_ETH) &&
    !(lookupChain === CHAIN_ID_TERRA); //Terra & ETH can proceed because it has advanced mode

  const content = fatalError ? (
    <Typography>{fatalError}</Typography>
  ) : lookupChain === CHAIN_ID_SOLANA ? (
    <SolanaSourceTokenSelector
      value={sourceParsedTokenAccount || null}
      onChange={handleSolanaOnChange}
      disabled={disabled}
      accounts={maps?.tokenAccounts?.data || []}
      solanaTokenMap={maps?.tokenMap}
      metaplexData={maps?.metaplex}
    />
  ) : lookupChain === CHAIN_ID_ETH ? (
    <EthereumSourceTokenSelector
      value={sourceParsedTokenAccount || null}
      disabled={disabled}
      onChange={handleSolanaOnChange}
      covalent={maps?.covalent || undefined}
      tokenAccounts={maps?.tokenAccounts} //TODO standardize
    />
  ) : lookupChain === CHAIN_ID_TERRA ? (
    <TerraSourceTokenSelector
      value={sourceParsedTokenAccount || null}
      disabled={disabled}
      onChange={handleSolanaOnChange}
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
