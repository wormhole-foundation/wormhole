import {
  CHAIN_ID_SOLANA,
  hexToNativeString,
  hexToUint8Array,
} from "@certusone/wormhole-sdk";
import { makeStyles, MenuItem, TextField, Typography } from "@material-ui/core";
import { Alert } from "@material-ui/lab";
import { PublicKey } from "@solana/web3.js";
import { BigNumber, ethers } from "ethers";
import { useCallback, useMemo } from "react";
import { useDispatch, useSelector } from "react-redux";
import { useBetaContext } from "../../contexts/BetaContext";
import useIsWalletReady from "../../hooks/useIsWalletReady";
import useSyncTargetAddress from "../../hooks/useSyncTargetAddress";
import { EthGasEstimateSummary } from "../../hooks/useTransactionFees";
import { incrementStep, setTargetChain } from "../../store/nftSlice";
import {
  selectNFTIsTargetComplete,
  selectNFTOriginAsset,
  selectNFTOriginChain,
  selectNFTOriginTokenId,
  selectNFTShouldLockFields,
  selectNFTSourceChain,
  selectNFTTargetAddressHex,
  selectNFTTargetAsset,
  selectNFTTargetBalanceString,
  selectNFTTargetChain,
  selectNFTTargetError,
} from "../../store/selectors";
import {
  BETA_CHAINS,
  CHAINS_BY_ID,
  CHAINS_WITH_NFT_SUPPORT,
} from "../../utils/consts";
import { isEVMChain } from "../../utils/ethereum";
import ButtonWithLoader from "../ButtonWithLoader";
import KeyAndBalance from "../KeyAndBalance";
import LowBalanceWarning from "../LowBalanceWarning";
import StepDescription from "../StepDescription";

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
  const isBeta = useBetaContext();
  const sourceChain = useSelector(selectNFTSourceChain);
  const chains = useMemo(
    () => CHAINS_WITH_NFT_SUPPORT.filter((c) => c.id !== sourceChain),
    [sourceChain]
  );
  const targetChain = useSelector(selectNFTTargetChain);
  const targetAddressHex = useSelector(selectNFTTargetAddressHex);
  const targetAsset = useSelector(selectNFTTargetAsset);
  const originChain = useSelector(selectNFTOriginChain);
  const originAsset = useSelector(selectNFTOriginAsset);
  const originTokenId = useSelector(selectNFTOriginTokenId);
  let tokenId;
  try {
    tokenId =
      originChain === CHAIN_ID_SOLANA && originAsset
        ? BigNumber.from(
            new PublicKey(hexToUint8Array(originAsset)).toBytes()
          ).toString()
        : originTokenId;
  } catch (e) {
    tokenId = originTokenId;
  }
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
        variant="outlined"
        value={targetChain}
        onChange={handleTargetChange}
      >
        {chains
          .filter(({ id }) => (isBeta ? true : !BETA_CHAINS.includes(id)))
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
        variant="outlined"
        className={classes.transferField}
        value={readableTargetAddress}
        disabled={true}
      />
      {targetAsset !== ethers.constants.AddressZero ? (
        <>
          <TextField
            label="Token Address"
            fullWidth
            variant="outlined"
            className={classes.transferField}
            value={targetAsset || ""}
            disabled={true}
          />
          {isEVMChain(targetChain) ? (
            <TextField
              variant="outlined"
              label="TokenId"
              fullWidth
              className={classes.transferField}
              value={tokenId || ""}
              disabled={true}
            />
          ) : null}
        </>
      ) : null}
      <Alert severity="info" variant="outlined" className={classes.alert}>
        <Typography>
          You will have to pay transaction fees on{" "}
          {CHAINS_BY_ID[targetChain].name} to redeem your NFT.
        </Typography>
        {isEVMChain(targetChain) && (
          <EthGasEstimateSummary methodType="nft" chainId={targetChain} />
        )}
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
