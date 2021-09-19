import { CHAIN_ID_ETH, CHAIN_ID_SOLANA } from "@certusone/wormhole-sdk";
import { Button, makeStyles, MenuItem, TextField } from "@material-ui/core";
import { Restore } from "@material-ui/icons";
import { useCallback } from "react";
import { useDispatch, useSelector } from "react-redux";
import useIsWalletReady from "../../hooks/useIsWalletReady";
import {
  selectNFTIsSourceComplete,
  selectNFTShouldLockFields,
  selectNFTSourceBalanceString,
  selectNFTSourceChain,
  selectNFTSourceError,
} from "../../store/selectors";
import { incrementStep, setSourceChain } from "../../store/nftSlice";
import { CHAINS } from "../../utils/consts";
import ButtonWithLoader from "../ButtonWithLoader";
import KeyAndBalance from "../KeyAndBalance";
import StepDescription from "../StepDescription";
import { TokenSelector } from "../TokenSelectors/SourceTokenSelector";
import { Alert } from "@material-ui/lab";
import LowBalanceWarning from "../LowBalanceWarning";

const useStyles = makeStyles((theme) => ({
  transferField: {
    marginTop: theme.spacing(5),
  },
}));

function Source({
  setIsRecoveryOpen,
}: {
  setIsRecoveryOpen: (open: boolean) => void;
}) {
  const classes = useStyles();
  const dispatch = useDispatch();
  const sourceChain = useSelector(selectNFTSourceChain);
  const uiAmountString = useSelector(selectNFTSourceBalanceString);
  const error = useSelector(selectNFTSourceError);
  const isSourceComplete = useSelector(selectNFTIsSourceComplete);
  const shouldLockFields = useSelector(selectNFTShouldLockFields);
  const { isReady, statusMessage } = useIsWalletReady(sourceChain);
  const handleSourceChange = useCallback(
    (event) => {
      dispatch(setSourceChain(event.target.value));
    },
    [dispatch]
  );
  const handleNextClick = useCallback(() => {
    dispatch(incrementStep());
  }, [dispatch]);
  return (
    <>
      <StepDescription>
        <div style={{ display: "flex", alignItems: "center" }}>
          Select an NFT to send through the Wormhole NFT Bridge.
          <div style={{ flexGrow: 1 }} />
          <Button
            onClick={() => setIsRecoveryOpen(true)}
            size="small"
            variant="outlined"
            endIcon={<Restore />}
          >
            Perform Recovery
          </Button>
        </div>
      </StepDescription>
      <TextField
        select
        fullWidth
        value={sourceChain}
        onChange={handleSourceChange}
        disabled={shouldLockFields}
      >
        {CHAINS.filter(
          ({ id }) => id === CHAIN_ID_ETH || id === CHAIN_ID_SOLANA
        ).map(({ id, name }) => (
          <MenuItem key={id} value={id}>
            {name}
          </MenuItem>
        ))}
      </TextField>
      {sourceChain === CHAIN_ID_ETH ? (
        <Alert severity="info">
          Only NFTs which implement ERC-721 are supported.
        </Alert>
      ) : null}
      <KeyAndBalance chainId={sourceChain} balance={uiAmountString} />
      {isReady || uiAmountString ? (
        <div className={classes.transferField}>
          <TokenSelector disabled={shouldLockFields} nft={true} />
        </div>
      ) : null}
      <LowBalanceWarning chainId={sourceChain} />
      <ButtonWithLoader
        disabled={!isSourceComplete}
        onClick={handleNextClick}
        showLoader={false}
        error={statusMessage || error}
      >
        Next
      </ButtonWithLoader>
    </>
  );
}

export default Source;
