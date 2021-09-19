import { CHAIN_ID_ETH, CHAIN_ID_SOLANA } from "@certusone/wormhole-sdk";
import { makeStyles, MenuItem, TextField } from "@material-ui/core";
import { useCallback, useMemo } from "react";
import { ethers } from "ethers";
import { useDispatch, useSelector } from "react-redux";
import useIsWalletReady from "../../hooks/useIsWalletReady";
import useSyncTargetAddress from "../../hooks/useSyncTargetAddress";
import { incrementStep, setTargetChain } from "../../store/nftSlice";
import {
  selectNFTIsTargetComplete,
  selectNFTOriginTokenId,
  selectNFTShouldLockFields,
  selectNFTSourceChain,
  selectNFTTargetAddressHex,
  selectNFTTargetAsset,
  selectNFTTargetBalanceString,
  selectNFTTargetChain,
  selectNFTTargetError,
} from "../../store/selectors";
import { hexToNativeString } from "../../utils/array";
import { CHAINS, CHAINS_BY_ID } from "../../utils/consts";
import ButtonWithLoader from "../ButtonWithLoader";
import KeyAndBalance from "../KeyAndBalance";
import StepDescription from "../StepDescription";
import LowBalanceWarning from "../LowBalanceWarning";
import { Alert } from "@material-ui/lab";

const useStyles = makeStyles((theme) => ({
  transferField: {
    marginTop: theme.spacing(5),
  },
  alert: {
    marginTop: theme.spacing(1),
    marginBottom: theme.spacing(1),
  },
}));

function Target() {
  const classes = useStyles();
  const dispatch = useDispatch();
  const sourceChain = useSelector(selectNFTSourceChain);
  const chains = useMemo(
    () => CHAINS.filter((c) => c.id !== sourceChain),
    [sourceChain]
  );
  const targetChain = useSelector(selectNFTTargetChain);
  const targetAddressHex = useSelector(selectNFTTargetAddressHex);
  const targetAsset = useSelector(selectNFTTargetAsset);
  const originTokenId = useSelector(selectNFTOriginTokenId);
  const readableTargetAddress =
    hexToNativeString(targetAddressHex, targetChain) || "";
  const uiAmountString = useSelector(selectNFTTargetBalanceString);
  const error = useSelector(selectNFTTargetError);
  const isTargetComplete = useSelector(selectNFTIsTargetComplete);
  const shouldLockFields = useSelector(selectNFTShouldLockFields);
  const { statusMessage } = useIsWalletReady(targetChain);
  useSyncTargetAddress(!shouldLockFields, true);
  const handleTargetChange = useCallback(
    (event) => {
      dispatch(setTargetChain(event.target.value));
    },
    [dispatch]
  );
  const handleNextClick = useCallback(() => {
    dispatch(incrementStep());
  }, [dispatch]);
  return (
    <>
      <StepDescription>Select a recipient chain and address.</StepDescription>
      <TextField
        select
        fullWidth
        value={targetChain}
        onChange={handleTargetChange}
        disabled={true}
      >
        {chains
          .filter(({ id }) => id === CHAIN_ID_ETH || id === CHAIN_ID_SOLANA)
          .map(({ id, name }) => (
            <MenuItem key={id} value={id}>
              {name}
            </MenuItem>
          ))}
      </TextField>
      <KeyAndBalance chainId={targetChain} balance={uiAmountString} />
      <TextField
        label="Recipient Address"
        fullWidth
        className={classes.transferField}
        value={readableTargetAddress}
        disabled={true}
      />
      {targetAsset !== ethers.constants.AddressZero ? (
        <>
          <TextField
            label="Token Address"
            fullWidth
            className={classes.transferField}
            value={targetAsset || ""}
            disabled={true}
          />
          {targetChain === CHAIN_ID_ETH ? (
            <TextField
              label="TokenId"
              fullWidth
              className={classes.transferField}
              value={originTokenId || ""}
              disabled={true}
            />
          ) : null}
        </>
      ) : null}
      <Alert severity="info" className={classes.alert}>
        You will have to pay transaction fees on{" "}
        {CHAINS_BY_ID[targetChain].name} to redeem your NFT.
      </Alert>
      <LowBalanceWarning chainId={targetChain} />
      <ButtonWithLoader
        disabled={!isTargetComplete} //|| !associatedAccountExists}
        onClick={handleNextClick}
        showLoader={false}
        error={statusMessage || error}
      >
        Next
      </ButtonWithLoader>
    </>
  );
}

export default Target;
