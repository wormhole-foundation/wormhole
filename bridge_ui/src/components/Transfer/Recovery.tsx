import {
  ChainId,
  CHAIN_ID_BSC,
  CHAIN_ID_ETH,
  CHAIN_ID_SOLANA,
  CHAIN_ID_TERRA,
  getEmitterAddressEth,
  getEmitterAddressSolana,
  getEmitterAddressTerra,
  parseSequenceFromLogEth,
  parseSequenceFromLogSolana,
  parseSequenceFromLogTerra,
} from "@certusone/wormhole-sdk";
import {
  Box,
  Button,
  CircularProgress,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Divider,
  Fab,
  makeStyles,
  MenuItem,
  TextField,
  Tooltip,
  Typography,
} from "@material-ui/core";
import { Restore } from "@material-ui/icons";
import { Alert } from "@material-ui/lab";
import { Connection } from "@solana/web3.js";
import { LCDClient } from "@terra-money/terra.js";
import { BigNumber, ethers } from "ethers";
import { useSnackbar } from "notistack";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useDispatch, useSelector } from "react-redux";
import { useEthereumProvider } from "../../contexts/EthereumProviderContext";
import {
  selectTransferSignedVAAHex,
  selectTransferSourceChain,
} from "../../store/selectors";
import {
  setSignedVAAHex,
  setStep,
  setTargetChain,
} from "../../store/transferSlice";
import {
  hexToNativeString,
  hexToUint8Array,
  uint8ArrayToHex,
} from "../../utils/array";
import {
  CHAINS,
  ETH_BRIDGE_ADDRESS,
  ETH_TOKEN_BRIDGE_ADDRESS,
  SOLANA_HOST,
  SOL_TOKEN_BRIDGE_ADDRESS,
  TERRA_HOST,
  TERRA_TOKEN_BRIDGE_ADDRESS,
  WORMHOLE_RPC_HOSTS,
} from "../../utils/consts";
import { getSignedVAAWithRetry } from "../../utils/getSignedVAAWithRetry";
import parseError from "../../utils/parseError";
import KeyAndBalance from "../KeyAndBalance";

const useStyles = makeStyles((theme) => ({
  fab: {
    position: "fixed",
    bottom: theme.spacing(2),
    right: theme.spacing(2),
  },
}));

async function eth(
  provider: ethers.providers.Web3Provider,
  tx: string,
  enqueueSnackbar: any
) {
  try {
    const receipt = await provider.getTransactionReceipt(tx);
    const sequence = parseSequenceFromLogEth(receipt, ETH_BRIDGE_ADDRESS);
    const emitterAddress = getEmitterAddressEth(ETH_TOKEN_BRIDGE_ADDRESS);
    const { vaaBytes } = await getSignedVAAWithRetry(
      CHAIN_ID_ETH,
      emitterAddress,
      sequence.toString(),
      WORMHOLE_RPC_HOSTS.length
    );
    return { vaa: uint8ArrayToHex(vaaBytes), error: null };
  } catch (e) {
    console.error(e);
    enqueueSnackbar(parseError(e), { variant: "error" });
    return { vaa: null, error: parseError(e) };
  }
}

async function solana(tx: string, enqueueSnackbar: any) {
  try {
    const connection = new Connection(SOLANA_HOST, "confirmed");
    const info = await connection.getTransaction(tx);
    if (!info) {
      throw new Error("An error occurred while fetching the transaction info");
    }
    const sequence = parseSequenceFromLogSolana(info);
    const emitterAddress = await getEmitterAddressSolana(
      SOL_TOKEN_BRIDGE_ADDRESS
    );
    const { vaaBytes } = await getSignedVAAWithRetry(
      CHAIN_ID_SOLANA,
      emitterAddress,
      sequence.toString(),
      WORMHOLE_RPC_HOSTS.length
    );
    return { vaa: uint8ArrayToHex(vaaBytes), error: null };
  } catch (e) {
    console.error(e);
    enqueueSnackbar(parseError(e), { variant: "error" });
    return { vaa: null, error: parseError(e) };
  }
}

async function terra(tx: string, enqueueSnackbar: any) {
  try {
    const lcd = new LCDClient(TERRA_HOST);
    const info = await lcd.tx.txInfo(tx);
    const sequence = parseSequenceFromLogTerra(info);
    if (!sequence) {
      throw new Error("Sequence not found");
    }
    const emitterAddress = await getEmitterAddressTerra(
      TERRA_TOKEN_BRIDGE_ADDRESS
    );
    const { vaaBytes } = await getSignedVAAWithRetry(
      CHAIN_ID_TERRA,
      emitterAddress,
      sequence,
      WORMHOLE_RPC_HOSTS.length
    );
    return { vaa: uint8ArrayToHex(vaaBytes), error: null };
  } catch (e) {
    console.error(e);
    enqueueSnackbar(parseError(e), { variant: "error" });
    return { vaa: null, error: parseError(e) };
  }
}

//     0   u256     amount
//     32  [u8; 32] token_address
//     64  u16      token_chain
//     66  [u8; 32] recipient
//     98  u16      recipient_chain
//     100 u256     fee

// TODO: move to wasm / sdk, share with solana
const parsePayload = (arr: Buffer) => ({
  amount: BigNumber.from(arr.slice(1, 1 + 32)).toBigInt(),
  originAddress: arr.slice(33, 33 + 32).toString("hex"),
  originChain: arr.readUInt16BE(65) as ChainId,
  targetAddress: arr.slice(67, 67 + 32).toString("hex"),
  targetChain: arr.readUInt16BE(99) as ChainId,
});

function RecoveryDialogContent({
  onClose,
  disabled,
}: {
  onClose: () => void;
  disabled: boolean;
}) {
  const { enqueueSnackbar } = useSnackbar();
  const dispatch = useDispatch();
  const { provider } = useEthereumProvider();
  const currentSourceChain = useSelector(selectTransferSourceChain);
  const [recoverySourceChain, setRecoverySourceChain] =
    useState(currentSourceChain);
  const [recoverySourceTx, setRecoverySourceTx] = useState("");
  const [recoverySourceTxIsLoading, setRecoverySourceTxIsLoading] =
    useState(false);
  const [recoverySourceTxError, setRecoverySourceTxError] = useState("");
  const currentSignedVAA = useSelector(selectTransferSignedVAAHex);
  const [recoverySignedVAA, setRecoverySignedVAA] = useState(currentSignedVAA);
  const [recoveryParsedVAA, setRecoveryParsedVAA] = useState<any>(null);
  useEffect(() => {
    if (!recoverySignedVAA) {
      setRecoverySourceTx("");
      setRecoverySourceChain(currentSourceChain);
    }
  }, [recoverySignedVAA, currentSourceChain]);
  useEffect(() => {
    if (recoverySourceTx) {
      let cancelled = false;
      if (recoverySourceChain === CHAIN_ID_ETH && provider) {
        setRecoverySourceTxError("");
        setRecoverySourceTxIsLoading(true);
        (async () => {
          const { vaa, error } = await eth(
            provider,
            recoverySourceTx,
            enqueueSnackbar
          );
          if (!cancelled) {
            setRecoverySourceTxIsLoading(false);
            if (vaa) {
              setRecoverySignedVAA(vaa);
            }
            if (error) {
              setRecoverySourceTxError(error);
            }
          }
        })();
      } else if (recoverySourceChain === CHAIN_ID_SOLANA) {
        setRecoverySourceTxError("");
        setRecoverySourceTxIsLoading(true);
        (async () => {
          const { vaa, error } = await solana(
            recoverySourceTx,
            enqueueSnackbar
          );
          if (!cancelled) {
            setRecoverySourceTxIsLoading(false);
            if (vaa) {
              setRecoverySignedVAA(vaa);
            }
            if (error) {
              setRecoverySourceTxError(error);
            }
          }
        })();
      } else if (recoverySourceChain === CHAIN_ID_TERRA) {
        setRecoverySourceTxError("");
        setRecoverySourceTxIsLoading(true);
        (async () => {
          const { vaa, error } = await terra(recoverySourceTx, enqueueSnackbar);
          if (!cancelled) {
            setRecoverySourceTxIsLoading(false);
            if (vaa) {
              setRecoverySignedVAA(vaa);
            }
            if (error) {
              setRecoverySourceTxError(error);
            }
          }
        })();
      }
      return () => {
        cancelled = true;
      };
    }
  }, [recoverySourceChain, recoverySourceTx, provider, enqueueSnackbar]);
  useEffect(() => {
    setRecoverySignedVAA(currentSignedVAA);
  }, [currentSignedVAA]);
  const handleSourceChainChange = useCallback((event) => {
    setRecoverySourceTx("");
    setRecoverySourceChain(event.target.value);
  }, []);
  const handleSourceTxChange = useCallback((event) => {
    setRecoverySourceTx(event.target.value.trim());
  }, []);
  const handleSignedVAAChange = useCallback((event) => {
    setRecoverySignedVAA(event.target.value.trim());
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
          in the Source Transaction ID (from Step 3) to resume your transfer.
        </Alert>
        <TextField
          select
          label="Source Chain"
          disabled={!!recoverySignedVAA}
          value={recoverySourceChain}
          onChange={handleSourceChainChange}
          fullWidth
          margin="normal"
        >
          {CHAINS.map(({ id, name }) => (
            <MenuItem key={id} value={id}>
              {name}
            </MenuItem>
          ))}
        </TextField>
        {recoverySourceChain === CHAIN_ID_ETH ||
        recoverySourceChain === CHAIN_ID_BSC ? (
          <KeyAndBalance chainId={recoverySourceChain} />
        ) : null}
        <TextField
          label="Source Tx (paste here)"
          disabled={!!recoverySignedVAA || recoverySourceTxIsLoading}
          value={recoverySourceTx}
          onChange={handleSourceTxChange}
          error={!!recoverySourceTxError}
          helperText={recoverySourceTxError}
          fullWidth
          margin="normal"
        />
        <Box position="relative">
          <Box mt={4}>
            <Typography>or</Typography>
          </Box>
          <TextField
            label="Signed VAA (Hex)"
            disabled={recoverySourceTxIsLoading}
            value={recoverySignedVAA || ""}
            onChange={handleSignedVAAChange}
            fullWidth
            margin="normal"
          />
          {recoverySourceTxIsLoading ? (
            <Box
              position="absolute"
              style={{
                top: 0,
                right: 0,
                left: 0,
                bottom: 0,
                backgroundColor: "rgba(0,0,0,0.5)",
                display: "flex",
                alignItems: "center",
                justifyContent: "center",
              }}
            >
              <CircularProgress />
            </Box>
          ) : null}
        </Box>
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
        <Button onClick={onClose} variant="outlined" color="default">
          Cancel
        </Button>
        <Button
          onClick={handleRecoverClick}
          variant="contained"
          color="primary"
          disabled={!enableRecovery || disabled}
        >
          Recover
        </Button>
      </DialogActions>
    </>
  );
}

export default function Recovery({
  open,
  setOpen,
  disabled,
}: {
  open: boolean;
  setOpen: (open: boolean) => void;
  disabled: boolean;
}) {
  const classes = useStyles();
  const handleOpenClick = useCallback(() => {
    setOpen(true);
  }, [setOpen]);
  const handleCloseClick = useCallback(() => {
    setOpen(false);
  }, [setOpen]);
  return (
    <>
      <Tooltip title="Open Recovery Dialog">
        <Fab className={classes.fab} onClick={handleOpenClick}>
          <Restore />
        </Fab>
      </Tooltip>
      <Dialog open={open} onClose={handleCloseClick} maxWidth="md" fullWidth>
        <DialogTitle>Recovery</DialogTitle>
        <RecoveryDialogContent onClose={handleCloseClick} disabled={disabled} />
      </Dialog>
    </>
  );
}
