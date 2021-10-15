import {
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Typography,
} from "@material-ui/core";
import { ArrowDownward } from "@material-ui/icons";
import { Alert } from "@material-ui/lab";
import { useSelector } from "react-redux";
import {
  selectTransferSourceChain,
  selectTransferSourceParsedTokenAccount,
} from "../../store/selectors";
import { CHAINS_BY_ID } from "../../utils/consts";
import SmartAddress from "../SmartAddress";
import { useTargetInfo } from "./Target";

function SendConfirmationContent() {
  const sourceChain = useSelector(selectTransferSourceChain);
  const sourceParsedTokenAccount = useSelector(
    selectTransferSourceParsedTokenAccount
  );
  const { targetChain, targetAsset, symbol, tokenName, logo } = useTargetInfo();
  return (
    <>
      {targetAsset ? (
        <div style={{ textAlign: "center" }}>
          <SmartAddress
            variant="h6"
            chainId={sourceChain}
            parsedTokenAccount={sourceParsedTokenAccount}
          />
          <div>
            <Typography variant="caption">
              {CHAINS_BY_ID[sourceChain].name}
            </Typography>
          </div>
          <div style={{ paddingTop: 4 }}>
            <ArrowDownward fontSize="inherit" />
          </div>
          <SmartAddress
            variant="h6"
            chainId={targetChain}
            address={targetAsset}
            symbol={symbol}
            tokenName={tokenName}
            logo={logo}
          />
          <div>
            <Typography variant="caption">
              {CHAINS_BY_ID[targetChain].name}
            </Typography>
          </div>
        </div>
      ) : null}
      <Alert severity="warning" variant="outlined" style={{ marginTop: 8 }}>
        Once the transfer transaction is submitted, the transfer must be
        completed by redeeming the tokens on the target chain. Please ensure
        that the token listed above is the desired token and confirm that
        markets exist on the target chain.
      </Alert>
    </>
  );
}

export default function SendConfirmationDialog({
  open,
  onClick,
  onClose,
}: {
  open: boolean;
  onClick: () => void;
  onClose: () => void;
}) {
  return (
    <Dialog open={open} onClose={onClose}>
      <DialogTitle>Are you sure?</DialogTitle>
      <DialogContent>
        <SendConfirmationContent />
      </DialogContent>
      <DialogActions>
        <Button variant="outlined" onClick={onClose}>
          Cancel
        </Button>
        <Button variant="contained" color="primary" onClick={onClick}>
          Confirm
        </Button>
      </DialogActions>
    </Dialog>
  );
}
