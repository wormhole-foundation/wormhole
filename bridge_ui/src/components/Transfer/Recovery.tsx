import {
  Box,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Divider,
  Fab,
  makeStyles,
  TextField,
} from "@material-ui/core";
import { Alert } from "@material-ui/lab";
import { Restore } from "@material-ui/icons";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useDispatch, useSelector } from "react-redux";
import {
  setSignedVAAHex,
  setStep,
  setTargetChain,
} from "../../store/transferSlice";
import { selectTransferSignedVAAHex } from "../../store/selectors";
import { hexToNativeString, hexToUint8Array } from "../../utils/array";
import { ChainId } from "@certusone/wormhole-sdk";
import { BigNumber } from "ethers";

const useStyles = makeStyles((theme) => ({
  fab: {
    position: "fixed",
    bottom: theme.spacing(2),
    right: theme.spacing(2),
  },
}));

//     0   u256     amount
//     32  [u8; 32] token_address
//     64  u16      token_chain
//     66  [u8; 32] recipient
//     98  u16      recipient_chain
//     100 u256     fee

// TODO: move to wasm / sdk, share with solana
const parsePayload = (arr: Buffer) => ({
  amount: BigNumber.from(arr.slice(1, 1 + 32)).toBigInt(),
  originAddress: arr.slice(33, 33 + 32).toString("hex"), // TODO: is this origin or source?
  originChain: arr.readUInt16BE(65) as ChainId, // TODO: is this origin or source?
  targetAddress: arr.slice(67, 67 + 32).toString("hex"),
  targetChain: arr.readUInt16BE(99) as ChainId,
});

function RecoveryDialogContent({ onClose }: { onClose: () => void }) {
  const dispatch = useDispatch();
  const currentSignedVAA = useSelector(selectTransferSignedVAAHex);
  const [recoverySignedVAA, setRecoverySignedVAA] = useState(currentSignedVAA);
  const [recoveryParsedVAA, setRecoveryParsedVAA] = useState<any>(null);
  const handleSignedVAAChange = useCallback((event) => {
    setRecoverySignedVAA(event.target.value);
  }, []);
  useEffect(() => {
    let cancelled = false;
    if (recoverySignedVAA) {
      (async () => {
        try {
          const { parse_vaa } = await import(
            "@certusone/wormhole-sdk/lib/solana/core/bridge"
          );
          const parsedVAA = parse_vaa(hexToUint8Array(recoverySignedVAA));
          if (!cancelled) {
            setRecoveryParsedVAA(parsedVAA);
          }
        } catch (e) {
          console.log(e);
          if (!cancelled) {
            setRecoveryParsedVAA(null);
          }
        }
      })();
    }
    return () => {
      cancelled = true;
    };
  }, [recoverySignedVAA]);
  const parsedPayload = useMemo(
    () =>
      recoveryParsedVAA?.payload
        ? parsePayload(Buffer.from(new Uint8Array(recoveryParsedVAA.payload)))
        : null,
    [recoveryParsedVAA]
  );
  const parsedPayloadTargetChain = parsedPayload?.targetChain;
  const enableRecovery = recoverySignedVAA && parsedPayloadTargetChain;
  const handleRecoverClick = useCallback(() => {
    if (enableRecovery && recoverySignedVAA && parsedPayloadTargetChain) {
      // TODO: make recovery reducer
      dispatch(setSignedVAAHex(recoverySignedVAA));
      dispatch(setTargetChain(parsedPayloadTargetChain));
      dispatch(setStep(3));
      onClose();
    }
  }, [
    dispatch,
    enableRecovery,
    recoverySignedVAA,
    parsedPayloadTargetChain,
    onClose,
  ]);
  return (
    <>
      <DialogContent>
        <Alert severity="info">
          If you have sent your tokens but have not redeemed them, you may paste
          the signed VAA here to resume from the redeem step.
        </Alert>
        <TextField
          label="Signed VAA (Hex)"
          value={recoverySignedVAA || ""}
          onChange={handleSignedVAAChange}
          fullWidth
          margin="normal"
        />
        <Box my={4}>
          <Divider />
        </Box>
        <TextField
          label="Emitter Chain"
          disabled
          value={recoveryParsedVAA?.emitter_chain || ""}
          fullWidth
          margin="normal"
        />
        <TextField
          label="Emitter Address"
          disabled
          value={
            (recoveryParsedVAA &&
              hexToNativeString(
                recoveryParsedVAA.emitter_address,
                recoveryParsedVAA.emitter_chain
              )) ||
            ""
          }
          fullWidth
          margin="normal"
        />
        <TextField
          label="Sequence"
          disabled
          value={recoveryParsedVAA?.sequence || ""}
          fullWidth
          margin="normal"
        />
        <TextField
          label="Timestamp"
          disabled
          value={
            (recoveryParsedVAA &&
              new Date(recoveryParsedVAA.timestamp * 1000).toLocaleString()) ||
            ""
          }
          fullWidth
          margin="normal"
        />
        <Box my={4}>
          <Divider />
        </Box>
        <TextField
          label="Origin Chain"
          disabled
          value={parsedPayload?.originChain.toString() || ""}
          fullWidth
          margin="normal"
        />
        <TextField
          label="Origin Token Address"
          disabled
          value={
            (parsedPayload &&
              hexToNativeString(
                parsedPayload.originAddress,
                parsedPayload.originChain
              )) ||
            ""
          }
          fullWidth
          margin="normal"
        />
        <TextField
          label="Target Chain"
          disabled
          value={parsedPayload?.targetChain.toString() || ""}
          fullWidth
          margin="normal"
        />
        <TextField
          label="Target Address"
          disabled
          value={
            (parsedPayload &&
              hexToNativeString(
                parsedPayload.targetAddress,
                parsedPayload.targetChain
              )) ||
            ""
          }
          fullWidth
          margin="normal"
        />
        <TextField
          label="Amount"
          disabled
          value={parsedPayload?.amount.toString() || ""}
          fullWidth
          margin="normal"
        />
        <Box my={4}>
          <Divider />
        </Box>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose} variant="contained" color="default">
          Cancel
        </Button>
        <Button
          onClick={handleRecoverClick}
          variant="contained"
          color="primary"
          disabled={!enableRecovery}
        >
          Recover
        </Button>
      </DialogActions>
    </>
  );
}

export default function Recovery() {
  const classes = useStyles();
  const [open, setOpen] = useState(false);
  const handleOpenClick = useCallback(() => {
    setOpen(true);
  }, []);
  const handleCloseClick = useCallback(() => {
    setOpen(false);
  }, []);
  return (
    <>
      <Fab className={classes.fab} onClick={handleOpenClick}>
        <Restore />
      </Fab>
      <Dialog open={open} onClose={handleCloseClick} maxWidth="md" fullWidth>
        <DialogTitle>Recovery</DialogTitle>
        <RecoveryDialogContent onClose={handleCloseClick} />
      </Dialog>
    </>
  );
}
